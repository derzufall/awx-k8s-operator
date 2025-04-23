package com.ansible.awx.operator.controller;

import com.ansible.awx.operator.client.AWXClient;
import com.ansible.awx.operator.client.AWXClientFactory;
import com.ansible.awx.operator.model.AWXConnectionConfig;
import com.ansible.awx.operator.model.AWXInstanceSpec;
import com.ansible.awx.operator.model.InventorySpec;
import com.ansible.awx.operator.model.JobTemplateSpec;
import com.ansible.awx.operator.model.ProjectSpec;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

/**
 * Controller for AWXInstance custom resources.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class AWXInstanceController {
    private final AWXClientFactory clientFactory;
    
    /**
     * Reconcile method for AWXInstance custom resources.
     */
    public void reconcile(String namespace, String name, AWXInstanceSpec spec, boolean isDeleted) {
        if (spec.getConnection() == null) {
            log.error("No connection configuration for AWXInstance {}/{}", namespace, name);
            return;
        }
        
        AWXClient awxClient = clientFactory.createClient(spec.getConnection());
        
        if (!awxClient.testConnection()) {
            log.error("Failed to connect to AWX at {}", spec.getConnection().getUrl());
            return;
        }
        
        if (isDeleted) {
            handleDeletion(namespace, name, spec, awxClient);
            return;
        }

        processResources(namespace, name, spec, awxClient);
    }
    
    private void processResources(String namespace, String name, AWXInstanceSpec spec, AWXClient awxClient) {
        // Process projects
        if (spec.getProjects() != null) {
            spec.getProjects().forEach(projectSpec -> 
                reconcileProject(namespace, projectSpec, awxClient));
        }
        
        // Process inventories
        if (spec.getInventories() != null) {
            spec.getInventories().forEach(inventorySpec -> 
                reconcileInventory(namespace, inventorySpec, awxClient));
        }
        
        // Process job templates
        if (spec.getJobTemplates() != null) {
            spec.getJobTemplates().forEach(jobTemplateSpec -> 
                reconcileJobTemplate(namespace, jobTemplateSpec, awxClient));
        }
    }
    
    private void handleDeletion(String namespace, String name, AWXInstanceSpec spec, AWXClient awxClient) {
        // Delete projects
        if (spec.getProjects() != null) {
            spec.getProjects().forEach(projectSpec -> 
                deleteProject(namespace, projectSpec, awxClient));
        }
        
        // Delete inventories
        if (spec.getInventories() != null) {
            spec.getInventories().forEach(inventorySpec -> 
                deleteInventory(namespace, inventorySpec, awxClient));
        }
        
        // Delete job templates
        if (spec.getJobTemplates() != null) {
            spec.getJobTemplates().forEach(jobTemplateSpec -> 
                deleteJobTemplate(namespace, jobTemplateSpec, awxClient));
        }
    }
    
    private void reconcileProject(String namespace, ProjectSpec spec, AWXClient awxClient) {
        try {
            var projectDto = awxClient.ensureProject(spec);
            if (projectDto != null) {
                log.debug("Reconciled project {}/{} with AWX ID: {}", 
                        namespace, spec.getName(), projectDto.getId());
            }
        } catch (Exception e) {
            log.error("Error reconciling project {}: {}", spec.getName(), e.getMessage());
        }
    }
    
    private void reconcileInventory(String namespace, InventorySpec spec, AWXClient awxClient) {
        try {
            var inventoryDto = awxClient.ensureInventory(spec);
            if (inventoryDto != null) {
                log.debug("Reconciled inventory {}/{} with AWX ID: {}", 
                        namespace, spec.getName(), inventoryDto.getId());
            }
        } catch (Exception e) {
            log.error("Error reconciling inventory {}: {}", spec.getName(), e.getMessage());
        }
    }
    
    private void reconcileJobTemplate(String namespace, JobTemplateSpec spec, AWXClient awxClient) {
        try {
            var jobTemplateDto = awxClient.ensureJobTemplate(spec);
            if (jobTemplateDto != null) {
                log.debug("Reconciled job template {}/{} with AWX ID: {}", 
                        namespace, spec.getName(), jobTemplateDto.getId());
            }
        } catch (Exception e) {
            log.error("Error reconciling job template {}: {}", spec.getName(), e.getMessage());
        }
    }
    
    private void deleteProject(String namespace, ProjectSpec spec, AWXClient awxClient) {
        try {
            if (awxClient.deleteProject(spec.getName())) {
                log.debug("Deleted project {}/{}", namespace, spec.getName());
            }
        } catch (Exception e) {
            log.error("Error deleting project {}: {}", spec.getName(), e.getMessage());
        }
    }
    
    private void deleteInventory(String namespace, InventorySpec spec, AWXClient awxClient) {
        try {
            if (awxClient.deleteInventory(spec.getName())) {
                log.debug("Deleted inventory {}/{}", namespace, spec.getName());
            }
        } catch (Exception e) {
            log.error("Error deleting inventory {}: {}", spec.getName(), e.getMessage());
        }
    }
    
    private void deleteJobTemplate(String namespace, JobTemplateSpec spec, AWXClient awxClient) {
        try {
            if (awxClient.deleteJobTemplate(spec.getName())) {
                log.debug("Deleted job template {}/{}", namespace, spec.getName());
            }
        } catch (Exception e) {
            log.error("Error deleting job template {}: {}", spec.getName(), e.getMessage());
        }
    }
} 