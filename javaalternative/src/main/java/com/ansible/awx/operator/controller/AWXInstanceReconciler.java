package com.ansible.awx.operator.controller;

import com.ansible.awx.operator.client.AWXClient;
import com.ansible.awx.operator.model.*;
import io.kubernetes.client.extended.controller.reconciler.Reconciler;
import io.kubernetes.client.extended.controller.reconciler.Request;
import io.kubernetes.client.extended.controller.reconciler.Result;
import io.kubernetes.client.informer.SharedIndexInformer;
import io.kubernetes.client.openapi.ApiException;
import io.kubernetes.client.openapi.apis.CustomObjectsApi;
import io.kubernetes.client.openapi.models.V1Condition;
import io.kubernetes.client.openapi.models.V1ObjectMeta;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.time.OffsetDateTime;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Reconciler for AWXInstance resources.
 */
@Component
@Slf4j
public class AWXInstanceReconciler implements Reconciler {
    private static final String AWX_FINALIZER = "awx.ansible.com/finalizer";
    private static final String GROUP = "awx.ansible.com";
    private static final String VERSION = "v1alpha1";
    private static final String PLURAL = "awxinstances";
    
    private final CustomObjectsApi customObjectsApi;
    private final SharedIndexInformer<AWXInstance> awxInstanceInformer;

    @Autowired
    public AWXInstanceReconciler(
            CustomObjectsApi customObjectsApi,
            SharedIndexInformer<AWXInstance> awxInstanceInformer) {
        this.customObjectsApi = customObjectsApi;
        this.awxInstanceInformer = awxInstanceInformer;
    }

    @Override
    public Result reconcile(Request request) {
        log.info("Reconciling AWXInstance {}/{}", request.getNamespace(), request.getName());
        
        try {
            Object apiObj = customObjectsApi.getNamespacedCustomObject(
                    GROUP, VERSION, request.getNamespace(), PLURAL, request.getName());
            
            if (apiObj == null) {
                log.info("AWXInstance {}/{} not found", request.getNamespace(), request.getName());
                return new Result(false);
            }
            
            // Convert to our model
            AWXInstance instance = awxInstanceInformer.getIndexer().getByKey(
                    request.getNamespace() + "/" + request.getName());
            
            if (instance == null) {
                log.info("AWXInstance {}/{} not found in cache", request.getNamespace(), request.getName());
                return new Result(true);
            }

            // Initialize status maps if they don't exist
            ensureStatusMapsInitialized(instance);
            
            // Check if this resource is being deleted
            if (instance.getMetadata().getDeletionTimestamp() != null) {
                return handleDeletion(instance);
            }
            
            // Add our finalizer if it doesn't exist
            if (!hasFinalizer(instance, AWX_FINALIZER)) {
                return addFinalizer(instance, AWX_FINALIZER);
            }
            
            // Create AWX client
            String protocol = instance.getSpec().getProtocol() != null ? 
                    instance.getSpec().getProtocol() : "https";
            String baseUrl = protocol + "://" + instance.getSpec().getHostname();
            AWXClient awxClient = new AWXClient(
                    baseUrl, 
                    instance.getSpec().getAdminUser(), 
                    instance.getSpec().getAdminPassword());
            
            // Check connection and update status
            boolean connected = checkConnection(instance, awxClient);
            
            // If external instance and connection failed, don't continue
            if (!connected && instance.getSpec().isExternalInstance()) {
                setReadyCondition(instance, false, "ConnectionFailed", 
                        "Failed to connect to external AWX instance");
                updateStatus(instance);
                return new Result(true, 30000L); // Retry after 30 seconds
            }
            
            // Reconcile internal changes
            boolean internalChanges = reconcileInternalChanges(instance, awxClient);
            if (internalChanges) {
                updateStatus(instance);
            }
            
            // Reconcile projects
            reconcileProjects(instance, awxClient);
            
            // Reconcile inventories
            reconcileInventories(instance, awxClient);
            
            // Reconcile job templates
            reconcileJobTemplates(instance, awxClient);
            
            // Set ready condition
            setReadyCondition(instance, true, "ReconciliationSucceeded", 
                    "AWXInstance resources have been reconciled successfully");
            
            // Update status
            updateStatus(instance);
            
            // Requeue after 30 seconds
            return new Result(true, 30000L);
        } catch (ApiException e) {
            log.error("Error reconciling AWXInstance {}/{}: {}", 
                    request.getNamespace(), request.getName(), e.getMessage());
            return new Result(true, 60000L); // Retry after 1 minute
        }
    }

    private void ensureStatusMapsInitialized(AWXInstance instance) {
        if (instance.getStatus() == null) {
            instance.setStatus(new AWXInstanceStatus());
        }
        
        if (instance.getStatus().getProjectStatuses() == null) {
            instance.getStatus().setProjectStatuses(new HashMap<>());
        }
        
        if (instance.getStatus().getInventoryStatuses() == null) {
            instance.getStatus().setInventoryStatuses(new HashMap<>());
        }
        
        if (instance.getStatus().getJobTemplateStatuses() == null) {
            instance.getStatus().setJobTemplateStatuses(new HashMap<>());
        }
        
        if (instance.getStatus().getConditions() == null) {
            instance.getStatus().setConditions(new ArrayList<>());
        }
    }

    private boolean hasFinalizer(AWXInstance instance, String finalizer) {
        if (instance.getMetadata().getFinalizers() == null) {
            return false;
        }
        return instance.getMetadata().getFinalizers().contains(finalizer);
    }

    private Result addFinalizer(AWXInstance instance, String finalizer) throws ApiException {
        log.info("Adding finalizer {} to AWXInstance {}/{}", 
                finalizer, instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        V1ObjectMeta metadata = instance.getMetadata();
        List<String> finalizers = metadata.getFinalizers();
        if (finalizers == null) {
            finalizers = new ArrayList<>();
        }
        finalizers.add(finalizer);
        metadata.setFinalizers(finalizers);
        
        customObjectsApi.replaceNamespacedCustomObject(
                GROUP, VERSION, metadata.getNamespace(), PLURAL, metadata.getName(), instance);
        
        return new Result(true);
    }

    private Result handleDeletion(AWXInstance instance) throws ApiException {
        log.info("Handling deletion of AWXInstance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        if (hasFinalizer(instance, AWX_FINALIZER)) {
            // Perform cleanup
            finalizeAWXInstance(instance);
            
            // Remove finalizer
            V1ObjectMeta metadata = instance.getMetadata();
            List<String> finalizers = metadata.getFinalizers();
            finalizers.remove(AWX_FINALIZER);
            metadata.setFinalizers(finalizers);
            
            customObjectsApi.replaceNamespacedCustomObject(
                    GROUP, VERSION, metadata.getNamespace(), PLURAL, metadata.getName(), instance);
        }
        
        return new Result(false);
    }

    private void finalizeAWXInstance(AWXInstance instance) {
        log.info("Finalizing AWXInstance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        try {
            // Create AWX client
            String protocol = instance.getSpec().getProtocol() != null ? 
                    instance.getSpec().getProtocol() : "https";
            String baseUrl = protocol + "://" + instance.getSpec().getHostname();
            AWXClient awxClient = new AWXClient(
                    baseUrl, 
                    instance.getSpec().getAdminUser(), 
                    instance.getSpec().getAdminPassword());
            
            // Delete job templates first (as they depend on projects and inventories)
            if (instance.getSpec().getJobTemplates() != null) {
                for (JobTemplateSpec jobTemplateSpec : instance.getSpec().getJobTemplates()) {
                    log.info("Deleting job template {}", jobTemplateSpec.getName());
                    awxClient.deleteJobTemplate(jobTemplateSpec.getName());
                }
            }
            
            // Delete inventories
            if (instance.getSpec().getInventories() != null) {
                for (InventorySpec inventorySpec : instance.getSpec().getInventories()) {
                    log.info("Deleting inventory {}", inventorySpec.getName());
                    awxClient.deleteInventory(inventorySpec.getName());
                }
            }
            
            // Delete projects
            if (instance.getSpec().getProjects() != null) {
                for (ProjectSpec projectSpec : instance.getSpec().getProjects()) {
                    log.info("Deleting project {}", projectSpec.getName());
                    awxClient.deleteProject(projectSpec.getName());
                }
            }
            
            log.info("Successfully finalized AWXInstance {}/{}", 
                    instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        } catch (Exception e) {
            log.error("Error finalizing AWXInstance {}/{}: {}", 
                    instance.getMetadata().getNamespace(), instance.getMetadata().getName(), 
                    e.getMessage());
        }
    }

    private boolean checkConnection(AWXInstance instance, AWXClient awxClient) {
        log.info("Checking connection to AWX instance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        try {
            // Update last connection check timestamp
            instance.getStatus().setLastConnectionCheck(OffsetDateTime.now());
            
            boolean connected = awxClient.testConnection();
            if (connected) {
                instance.getStatus().setConnectionStatus("Connected");
                log.info("Successfully connected to AWX instance {}/{}", 
                        instance.getMetadata().getNamespace(), instance.getMetadata().getName());
            } else {
                instance.getStatus().setConnectionStatus("Failed: Connection test failed");
                log.error("Failed to connect to AWX instance {}/{}", 
                        instance.getMetadata().getNamespace(), instance.getMetadata().getName());
            }
            
            return connected;
        } catch (Exception e) {
            instance.getStatus().setConnectionStatus("Failed: " + e.getMessage());
            log.error("Error connecting to AWX instance {}/{}: {}", 
                    instance.getMetadata().getNamespace(), instance.getMetadata().getName(), 
                    e.getMessage());
            return false;
        }
    }

    private boolean reconcileInternalChanges(AWXInstance instance, AWXClient awxClient) {
        log.info("Reconciling internal changes for AWXInstance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        boolean changesDetected = false;
        
        // Check projects
        if (instance.getSpec().getProjects() != null) {
            for (ProjectSpec projectSpec : instance.getSpec().getProjects()) {
                log.info("Checking project state {}", projectSpec.getName());
                
                Map<String, Object> project = awxClient.getProject(projectSpec.getName());
                if (project == null) {
                    log.info("Project {} needs reconciliation (does not exist)", projectSpec.getName());
                    awxClient.ensureProject(projectSpec);
                    instance.getStatus().getProjectStatuses().put(
                            projectSpec.getName(), "Reconciled (corrected internal changes)");
                    changesDetected = true;
                }
                // Note: In a full implementation, we would also check if the project configuration matches
                // the desired state and update it if needed.
            }
        }
        
        // Check inventories
        if (instance.getSpec().getInventories() != null) {
            for (InventorySpec inventorySpec : instance.getSpec().getInventories()) {
                log.info("Checking inventory state {}", inventorySpec.getName());
                
                Map<String, Object> inventory = awxClient.getInventory(inventorySpec.getName());
                if (inventory == null) {
                    log.info("Inventory {} needs reconciliation (does not exist)", inventorySpec.getName());
                    awxClient.ensureInventory(inventorySpec);
                    instance.getStatus().getInventoryStatuses().put(
                            inventorySpec.getName(), "Reconciled (corrected internal changes)");
                    changesDetected = true;
                }
                // Note: In a full implementation, we would also check if the inventory configuration matches
                // the desired state and update it if needed.
            }
        }
        
        // Check job templates
        if (instance.getSpec().getJobTemplates() != null) {
            for (JobTemplateSpec jobTemplateSpec : instance.getSpec().getJobTemplates()) {
                log.info("Checking job template state {}", jobTemplateSpec.getName());
                
                Map<String, Object> jobTemplate = awxClient.getJobTemplate(jobTemplateSpec.getName());
                if (jobTemplate == null) {
                    log.info("Job template {} needs reconciliation (does not exist)", jobTemplateSpec.getName());
                    awxClient.ensureJobTemplate(jobTemplateSpec);
                    instance.getStatus().getJobTemplateStatuses().put(
                            jobTemplateSpec.getName(), "Reconciled (corrected internal changes)");
                    changesDetected = true;
                }
                // Note: In a full implementation, we would also check if the job template configuration matches
                // the desired state and update it if needed.
            }
        }
        
        return changesDetected;
    }

    private void reconcileProjects(AWXInstance instance, AWXClient awxClient) {
        log.info("Reconciling projects for AWXInstance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        if (instance.getSpec().getProjects() != null) {
            for (ProjectSpec projectSpec : instance.getSpec().getProjects()) {
                log.info("Reconciling project {}", projectSpec.getName());
                
                try {
                    awxClient.ensureProject(projectSpec);
                    instance.getStatus().getProjectStatuses().put(projectSpec.getName(), "Reconciled");
                } catch (Exception e) {
                    log.error("Failed to reconcile project {}: {}", 
                            projectSpec.getName(), e.getMessage());
                    instance.getStatus().getProjectStatuses().put(
                            projectSpec.getName(), "Failed: " + e.getMessage());
                }
            }
        }
    }

    private void reconcileInventories(AWXInstance instance, AWXClient awxClient) {
        log.info("Reconciling inventories for AWXInstance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        if (instance.getSpec().getInventories() != null) {
            for (InventorySpec inventorySpec : instance.getSpec().getInventories()) {
                log.info("Reconciling inventory {}", inventorySpec.getName());
                
                try {
                    awxClient.ensureInventory(inventorySpec);
                    instance.getStatus().getInventoryStatuses().put(inventorySpec.getName(), "Reconciled");
                } catch (Exception e) {
                    log.error("Failed to reconcile inventory {}: {}", 
                            inventorySpec.getName(), e.getMessage());
                    instance.getStatus().getInventoryStatuses().put(
                            inventorySpec.getName(), "Failed: " + e.getMessage());
                }
            }
        }
    }

    private void reconcileJobTemplates(AWXInstance instance, AWXClient awxClient) {
        log.info("Reconciling job templates for AWXInstance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        if (instance.getSpec().getJobTemplates() != null) {
            for (JobTemplateSpec jobTemplateSpec : instance.getSpec().getJobTemplates()) {
                log.info("Reconciling job template {}", jobTemplateSpec.getName());
                
                try {
                    awxClient.ensureJobTemplate(jobTemplateSpec);
                    instance.getStatus().getJobTemplateStatuses().put(jobTemplateSpec.getName(), "Reconciled");
                } catch (Exception e) {
                    log.error("Failed to reconcile job template {}: {}", 
                            jobTemplateSpec.getName(), e.getMessage());
                    instance.getStatus().getJobTemplateStatuses().put(
                            jobTemplateSpec.getName(), "Failed: " + e.getMessage());
                }
            }
        }
    }

    private void setReadyCondition(AWXInstance instance, boolean ready, String reason, String message) {
        V1Condition readyCondition = new V1Condition();
        readyCondition.setType("Ready");
        readyCondition.setStatus(ready ? "True" : "False");
        readyCondition.setLastTransitionTime(OffsetDateTime.now());
        readyCondition.setReason(reason);
        readyCondition.setMessage(message);
        
        List<V1Condition> conditions = instance.getStatus().getConditions();
        conditions.removeIf(c -> "Ready".equals(c.getType()));
        conditions.add(readyCondition);
    }

    private void updateStatus(AWXInstance instance) throws ApiException {
        log.info("Updating status for AWXInstance {}/{}", 
                instance.getMetadata().getNamespace(), instance.getMetadata().getName());
        
        Map<String, Object> status = new HashMap<>();
        status.put("conditions", instance.getStatus().getConditions());
        status.put("projectStatuses", instance.getStatus().getProjectStatuses());
        status.put("inventoryStatuses", instance.getStatus().getInventoryStatuses());
        status.put("jobTemplateStatuses", instance.getStatus().getJobTemplateStatuses());
        status.put("lastConnectionCheck", instance.getStatus().getLastConnectionCheck());
        status.put("connectionStatus", instance.getStatus().getConnectionStatus());
        
        Map<String, Object> body = new HashMap<>();
        body.put("status", status);
        
        customObjectsApi.patchNamespacedCustomObjectStatus(
                GROUP, VERSION, 
                instance.getMetadata().getNamespace(), PLURAL, 
                instance.getMetadata().getName(), 
                body);
    }
} 