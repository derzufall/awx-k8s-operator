package com.ansible.awx.operator.service;

import com.ansible.awx.operator.client.AWXClient;
import com.ansible.awx.operator.model.AWXInstance;
import io.kubernetes.client.informer.SharedIndexInformer;
import io.kubernetes.client.openapi.ApiException;
import io.kubernetes.client.openapi.apis.CustomObjectsApi;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.time.OffsetDateTime;
import java.util.HashMap;
import java.util.Map;

/**
 * Service that proactively monitors AWX instance connections.
 */
@Service
@Slf4j
public class ConnectionMonitorService {
    private static final String GROUP = "awx.ansible.com";
    private static final String VERSION = "v1alpha1";
    private static final String PLURAL = "awxinstances";
    
    private final CustomObjectsApi customObjectsApi;
    private final SharedIndexInformer<AWXInstance> awxInstanceInformer;

    @Autowired
    public ConnectionMonitorService(
            CustomObjectsApi customObjectsApi,
            SharedIndexInformer<AWXInstance> awxInstanceInformer) {
        this.customObjectsApi = customObjectsApi;
        this.awxInstanceInformer = awxInstanceInformer;
    }

    /**
     * Scheduled task that runs every 60 seconds to check connections to all AWX instances.
     * This provides a proactive way to detect connectivity issues outside the normal reconciliation loop.
     */
    @Scheduled(fixedRate = 60000)
    public void monitorConnections() {
        log.debug("Running scheduled connection check for all AWX instances");
        
        try {
            // Get all AWX instances from the informer cache
            awxInstanceInformer.getIndexer().list().forEach(instance -> {
                try {
                    checkInstanceConnection(instance);
                } catch (Exception e) {
                    log.error("Error checking connection for instance {}/{}: {}",
                            instance.getMetadata().getNamespace(),
                            instance.getMetadata().getName(),
                            e.getMessage());
                }
            });
        } catch (Exception e) {
            log.error("Error during scheduled connection monitoring: {}", e.getMessage());
        }
    }
    
    private void checkInstanceConnection(AWXInstance instance) {
        log.debug("Checking connection for AWX instance {}/{}",
                instance.getMetadata().getNamespace(), 
                instance.getMetadata().getName());
        
        try {
            // Create AWX client
            String protocol = instance.getSpec().getProtocol() != null ? 
                    instance.getSpec().getProtocol() : "https";
            String baseUrl = protocol + "://" + instance.getSpec().getHostname();
            AWXClient awxClient = new AWXClient(
                    baseUrl, 
                    instance.getSpec().getAdminUser(), 
                    instance.getSpec().getAdminPassword());
            
            // Test connection
            boolean connected = awxClient.testConnection();
            
            // Prepare status update
            Map<String, Object> statusPatch = new HashMap<>();
            Map<String, Object> status = new HashMap<>();
            status.put("lastConnectionCheck", OffsetDateTime.now());
            status.put("connectionStatus", connected ? "Connected" : "Failed: Connection test failed");
            statusPatch.put("status", status);
            
            // Update status
            customObjectsApi.patchNamespacedCustomObjectStatus(
                    GROUP, VERSION,
                    instance.getMetadata().getNamespace(),
                    PLURAL,
                    instance.getMetadata().getName(),
                    statusPatch);
            
            // Log result
            if (connected) {
                log.debug("Connection check successful for AWX instance {}/{}",
                        instance.getMetadata().getNamespace(),
                        instance.getMetadata().getName());
            } else {
                log.warn("Connection check failed for AWX instance {}/{}",
                        instance.getMetadata().getNamespace(),
                        instance.getMetadata().getName());
            }
        } catch (ApiException e) {
            log.error("API error updating connection status for AWX instance {}/{}: {}",
                    instance.getMetadata().getNamespace(),
                    instance.getMetadata().getName(),
                    e.getMessage());
        }
    }
} 