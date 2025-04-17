package com.ansible.awx.operator.client;

import com.ansible.awx.operator.model.InventorySpec;
import com.ansible.awx.operator.model.JobTemplateSpec;
import com.ansible.awx.operator.model.ProjectSpec;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.*;
import org.springframework.web.client.RestTemplate;

import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.Collections;
import java.util.Map;

/**
 * Client for interacting with the AWX API.
 */
@Slf4j
public class AWXClient {
    private final String baseUrl;
    private final String username;
    private final String password;
    private final RestTemplate restTemplate;
    private final ObjectMapper objectMapper;

    public AWXClient(String baseUrl, String username, String password) {
        this.baseUrl = baseUrl;
        this.username = username;
        this.password = password;
        this.restTemplate = new RestTemplate();
        this.objectMapper = new ObjectMapper();
    }

    /**
     * Tests the connection to the AWX instance.
     */
    public boolean testConnection() {
        try {
            HttpHeaders headers = createHeaders();
            HttpEntity<Void> entity = new HttpEntity<>(headers);
            
            ResponseEntity<String> response = restTemplate.exchange(
                    baseUrl + "/api/v2/ping/", 
                    HttpMethod.GET, 
                    entity, 
                    String.class);
            
            return response.getStatusCode().is2xxSuccessful();
        } catch (Exception e) {
            log.error("Failed to connect to AWX instance: {}", e.getMessage());
            return false;
        }
    }

    /**
     * Gets a project by name.
     */
    public Map<String, Object> getProject(String name) {
        try {
            HttpHeaders headers = createHeaders();
            HttpEntity<Void> entity = new HttpEntity<>(headers);
            
            ResponseEntity<String> response = restTemplate.exchange(
                    baseUrl + "/api/v2/projects/?name=" + name,
                    HttpMethod.GET,
                    entity,
                    String.class);
            
            JsonNode root = objectMapper.readTree(response.getBody());
            if (root.has("results") && root.get("results").size() > 0) {
                return objectMapper.convertValue(root.get("results").get(0), Map.class);
            }
            return null;
        } catch (Exception e) {
            log.error("Failed to get project {}: {}", name, e.getMessage());
            return null;
        }
    }

    /**
     * Ensures a project exists with the specified configuration.
     */
    public Map<String, Object> ensureProject(ProjectSpec projectSpec) {
        Map<String, Object> project = getProject(projectSpec.getName());
        
        ObjectNode projectData = objectMapper.createObjectNode()
                .put("name", projectSpec.getName())
                .put("description", projectSpec.getDescription())
                .put("scm_type", projectSpec.getScmType());
                
        if (projectSpec.getScmType() != null && !projectSpec.getScmType().equals("manual") && 
                projectSpec.getScmUrl() != null) {
            projectData.put("scm_url", projectSpec.getScmUrl());
        }
        
        if (projectSpec.getScmBranch() != null) {
            projectData.put("scm_branch", projectSpec.getScmBranch());
        }
        
        // Note: SCM credential handling would go here
        
        try {
            HttpHeaders headers = createHeaders();
            headers.setContentType(MediaType.APPLICATION_JSON);
            
            HttpEntity<String> entity = new HttpEntity<>(objectMapper.writeValueAsString(projectData), headers);
            
            if (project == null) {
                // Create project
                ResponseEntity<String> response = restTemplate.exchange(
                        baseUrl + "/api/v2/projects/",
                        HttpMethod.POST,
                        entity,
                        String.class);
                
                return objectMapper.readValue(response.getBody(), Map.class);
            } else {
                // Update project
                ResponseEntity<String> response = restTemplate.exchange(
                        baseUrl + "/api/v2/projects/" + project.get("id") + "/",
                        HttpMethod.PUT,
                        entity,
                        String.class);
                
                return objectMapper.readValue(response.getBody(), Map.class);
            }
        } catch (Exception e) {
            log.error("Failed to ensure project {}: {}", projectSpec.getName(), e.getMessage());
            return null;
        }
    }

    /**
     * Gets an inventory by name.
     */
    public Map<String, Object> getInventory(String name) {
        try {
            HttpHeaders headers = createHeaders();
            HttpEntity<Void> entity = new HttpEntity<>(headers);
            
            ResponseEntity<String> response = restTemplate.exchange(
                    baseUrl + "/api/v2/inventories/?name=" + name,
                    HttpMethod.GET,
                    entity,
                    String.class);
            
            JsonNode root = objectMapper.readTree(response.getBody());
            if (root.has("results") && root.get("results").size() > 0) {
                return objectMapper.convertValue(root.get("results").get(0), Map.class);
            }
            return null;
        } catch (Exception e) {
            log.error("Failed to get inventory {}: {}", name, e.getMessage());
            return null;
        }
    }

    /**
     * Ensures an inventory exists with the specified configuration.
     */
    public Map<String, Object> ensureInventory(InventorySpec inventorySpec) {
        Map<String, Object> inventory = getInventory(inventorySpec.getName());
        
        ObjectNode inventoryData = objectMapper.createObjectNode()
                .put("name", inventorySpec.getName())
                .put("description", inventorySpec.getDescription());
                
        if (inventorySpec.getVariables() != null) {
            inventoryData.put("variables", inventorySpec.getVariables());
        }
        
        try {
            HttpHeaders headers = createHeaders();
            headers.setContentType(MediaType.APPLICATION_JSON);
            
            HttpEntity<String> entity = new HttpEntity<>(objectMapper.writeValueAsString(inventoryData), headers);
            
            if (inventory == null) {
                // Create inventory
                ResponseEntity<String> response = restTemplate.exchange(
                        baseUrl + "/api/v2/inventories/",
                        HttpMethod.POST,
                        entity,
                        String.class);
                
                inventory = objectMapper.readValue(response.getBody(), Map.class);
            } else {
                // Update inventory
                ResponseEntity<String> response = restTemplate.exchange(
                        baseUrl + "/api/v2/inventories/" + inventory.get("id") + "/",
                        HttpMethod.PUT,
                        entity,
                        String.class);
                
                inventory = objectMapper.readValue(response.getBody(), Map.class);
            }
            
            // TODO: Handle hosts if needed
            
            return inventory;
        } catch (Exception e) {
            log.error("Failed to ensure inventory {}: {}", inventorySpec.getName(), e.getMessage());
            return null;
        }
    }

    /**
     * Gets a job template by name.
     */
    public Map<String, Object> getJobTemplate(String name) {
        try {
            HttpHeaders headers = createHeaders();
            HttpEntity<Void> entity = new HttpEntity<>(headers);
            
            ResponseEntity<String> response = restTemplate.exchange(
                    baseUrl + "/api/v2/job_templates/?name=" + name,
                    HttpMethod.GET,
                    entity,
                    String.class);
            
            JsonNode root = objectMapper.readTree(response.getBody());
            if (root.has("results") && root.get("results").size() > 0) {
                return objectMapper.convertValue(root.get("results").get(0), Map.class);
            }
            return null;
        } catch (Exception e) {
            log.error("Failed to get job template {}: {}", name, e.getMessage());
            return null;
        }
    }

    /**
     * Ensures a job template exists with the specified configuration.
     */
    public Map<String, Object> ensureJobTemplate(JobTemplateSpec jobTemplateSpec) {
        Map<String, Object> jobTemplate = getJobTemplate(jobTemplateSpec.getName());
        
        // Find project and inventory
        Map<String, Object> project = getProject(jobTemplateSpec.getProjectName());
        if (project == null) {
            log.error("Project {} not found", jobTemplateSpec.getProjectName());
            return null;
        }
        
        Map<String, Object> inventory = getInventory(jobTemplateSpec.getInventoryName());
        if (inventory == null) {
            log.error("Inventory {} not found", jobTemplateSpec.getInventoryName());
            return null;
        }
        
        ObjectNode jobTemplateData = objectMapper.createObjectNode()
                .put("name", jobTemplateSpec.getName())
                .put("description", jobTemplateSpec.getDescription())
                .put("project", Integer.parseInt(project.get("id").toString()))
                .put("inventory", Integer.parseInt(inventory.get("id").toString()))
                .put("playbook", jobTemplateSpec.getPlaybook());
                
        if (jobTemplateSpec.getExtraVars() != null) {
            jobTemplateData.put("extra_vars", jobTemplateSpec.getExtraVars());
        }
        
        try {
            HttpHeaders headers = createHeaders();
            headers.setContentType(MediaType.APPLICATION_JSON);
            
            HttpEntity<String> entity = new HttpEntity<>(objectMapper.writeValueAsString(jobTemplateData), headers);
            
            if (jobTemplate == null) {
                // Create job template
                ResponseEntity<String> response = restTemplate.exchange(
                        baseUrl + "/api/v2/job_templates/",
                        HttpMethod.POST,
                        entity,
                        String.class);
                
                return objectMapper.readValue(response.getBody(), Map.class);
            } else {
                // Update job template
                ResponseEntity<String> response = restTemplate.exchange(
                        baseUrl + "/api/v2/job_templates/" + jobTemplate.get("id") + "/",
                        HttpMethod.PUT,
                        entity,
                        String.class);
                
                return objectMapper.readValue(response.getBody(), Map.class);
            }
        } catch (Exception e) {
            log.error("Failed to ensure job template {}: {}", jobTemplateSpec.getName(), e.getMessage());
            return null;
        }
    }

    /**
     * Delete a project by name.
     */
    public boolean deleteProject(String name) {
        Map<String, Object> project = getProject(name);
        if (project == null) {
            return true; // Already gone
        }
        
        try {
            HttpHeaders headers = createHeaders();
            HttpEntity<Void> entity = new HttpEntity<>(headers);
            
            ResponseEntity<Void> response = restTemplate.exchange(
                    baseUrl + "/api/v2/projects/" + project.get("id") + "/",
                    HttpMethod.DELETE,
                    entity,
                    Void.class);
            
            return response.getStatusCode().is2xxSuccessful();
        } catch (Exception e) {
            log.error("Failed to delete project {}: {}", name, e.getMessage());
            return false;
        }
    }

    /**
     * Delete an inventory by name.
     */
    public boolean deleteInventory(String name) {
        Map<String, Object> inventory = getInventory(name);
        if (inventory == null) {
            return true; // Already gone
        }
        
        try {
            HttpHeaders headers = createHeaders();
            HttpEntity<Void> entity = new HttpEntity<>(headers);
            
            ResponseEntity<Void> response = restTemplate.exchange(
                    baseUrl + "/api/v2/inventories/" + inventory.get("id") + "/",
                    HttpMethod.DELETE,
                    entity,
                    Void.class);
            
            return response.getStatusCode().is2xxSuccessful();
        } catch (Exception e) {
            log.error("Failed to delete inventory {}: {}", name, e.getMessage());
            return false;
        }
    }

    /**
     * Delete a job template by name.
     */
    public boolean deleteJobTemplate(String name) {
        Map<String, Object> jobTemplate = getJobTemplate(name);
        if (jobTemplate == null) {
            return true; // Already gone
        }
        
        try {
            HttpHeaders headers = createHeaders();
            HttpEntity<Void> entity = new HttpEntity<>(headers);
            
            ResponseEntity<Void> response = restTemplate.exchange(
                    baseUrl + "/api/v2/job_templates/" + jobTemplate.get("id") + "/",
                    HttpMethod.DELETE,
                    entity,
                    Void.class);
            
            return response.getStatusCode().is2xxSuccessful();
        } catch (Exception e) {
            log.error("Failed to delete job template {}: {}", name, e.getMessage());
            return false;
        }
    }

    /**
     * Creates headers with basic auth for AWX API requests.
     */
    private HttpHeaders createHeaders() {
        HttpHeaders headers = new HttpHeaders();
        String auth = username + ":" + password;
        byte[] encodedAuth = Base64.getEncoder().encode(auth.getBytes(StandardCharsets.UTF_8));
        String authHeader = "Basic " + new String(encodedAuth);
        headers.set("Authorization", authHeader);
        headers.setAccept(Collections.singletonList(MediaType.APPLICATION_JSON));
        return headers;
    }
} 