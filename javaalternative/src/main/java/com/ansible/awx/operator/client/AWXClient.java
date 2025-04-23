package com.ansible.awx.operator.client;

import com.ansible.awx.operator.client.dto.InventoryDto;
import com.ansible.awx.operator.client.dto.JobTemplateDto;
import com.ansible.awx.operator.client.dto.ListResponse;
import com.ansible.awx.operator.client.dto.PingResponse;
import com.ansible.awx.operator.client.dto.ProjectDto;
import com.ansible.awx.operator.model.InventorySpec;
import com.ansible.awx.operator.model.JobTemplateSpec;
import com.ansible.awx.operator.model.ProjectSpec;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

/**
 * Client for interacting with the AWX API.
 */
@Slf4j
@RequiredArgsConstructor
public class AWXClient {
    private final AWXApiClient apiClient;

    public boolean testConnection() {
        try {
            PingResponse response = apiClient.ping();
            return response.getOnline() != null && response.getOnline();
        } catch (Exception e) {
            log.error("Connection test failed: {}", e.getMessage());
            return false;
        }
    }

    public ProjectDto getProject(String name) {
        try {
            ListResponse<ProjectDto> response = apiClient.getProjects(name);
            if (response != null && response.getResults() != null && !response.getResults().isEmpty()) {
                return response.getResults().get(0);
            }
            return null;
        } catch (Exception e) {
            log.error("Get project failed: {}", e.getMessage());
            return null;
        }
    }

    public ProjectDto ensureProject(ProjectSpec projectSpec) {
        ProjectDto project = getProject(projectSpec.getName());
        
        ProjectDto projectDto = ProjectDto.builder()
                .name(projectSpec.getName())
                .description(projectSpec.getDescription())
                .scmType(projectSpec.getScmType())
                .build();
                
        if (projectSpec.getScmType() != null && !projectSpec.getScmType().equals("manual") && 
                projectSpec.getScmUrl() != null) {
            projectDto.setScmUrl(projectSpec.getScmUrl());
        }
        
        if (projectSpec.getScmBranch() != null) {
            projectDto.setScmBranch(projectSpec.getScmBranch());
        }
        
        try {
            if (project == null) {
                return apiClient.createProject(projectDto);
            } else {
                return apiClient.updateProject(project.getId(), projectDto);
            }
        } catch (Exception e) {
            log.error("Ensure project failed: {}", e.getMessage());
            return null;
        }
    }

    public InventoryDto getInventory(String name) {
        try {
            ListResponse<InventoryDto> response = apiClient.getInventories(name);
            if (response != null && response.getResults() != null && !response.getResults().isEmpty()) {
                return response.getResults().get(0);
            }
            return null;
        } catch (Exception e) {
            log.error("Get inventory failed: {}", e.getMessage());
            return null;
        }
    }

    public InventoryDto ensureInventory(InventorySpec inventorySpec) {
        InventoryDto inventory = getInventory(inventorySpec.getName());
        
        InventoryDto inventoryDto = InventoryDto.builder()
                .name(inventorySpec.getName())
                .description(inventorySpec.getDescription())
                .variables(inventorySpec.getVariables())
                .build();
        
        try {
            if (inventory == null) {
                inventory = apiClient.createInventory(inventoryDto);
            } else {
                inventory = apiClient.updateInventory(inventory.getId(), inventoryDto);
            }
            
            return inventory;
        } catch (Exception e) {
            log.error("Ensure inventory failed: {}", e.getMessage());
            return null;
        }
    }

    public JobTemplateDto getJobTemplate(String name) {
        try {
            ListResponse<JobTemplateDto> response = apiClient.getJobTemplates(name);
            if (response != null && response.getResults() != null && !response.getResults().isEmpty()) {
                return response.getResults().get(0);
            }
            return null;
        } catch (Exception e) {
            log.error("Get job template failed: {}", e.getMessage());
            return null;
        }
    }

    public JobTemplateDto ensureJobTemplate(JobTemplateSpec jobTemplateSpec) {
        JobTemplateDto jobTemplate = getJobTemplate(jobTemplateSpec.getName());
        
        ProjectDto project = getProject(jobTemplateSpec.getProjectName());
        if (project == null) {
            log.error("Project {} not found", jobTemplateSpec.getProjectName());
            return null;
        }
        
        InventoryDto inventory = getInventory(jobTemplateSpec.getInventoryName());
        if (inventory == null) {
            log.error("Inventory {} not found", jobTemplateSpec.getInventoryName());
            return null;
        }
        
        JobTemplateDto jobTemplateDto = JobTemplateDto.builder()
                .name(jobTemplateSpec.getName())
                .description(jobTemplateSpec.getDescription())
                .project(project.getId())
                .inventory(inventory.getId())
                .playbook(jobTemplateSpec.getPlaybook())
                .extraVars(jobTemplateSpec.getExtraVars())
                .build();
        
        try {
            if (jobTemplate == null) {
                return apiClient.createJobTemplate(jobTemplateDto);
            } else {
                return apiClient.updateJobTemplate(jobTemplate.getId(), jobTemplateDto);
            }
        } catch (Exception e) {
            log.error("Ensure job template failed: {}", e.getMessage());
            return null;
        }
    }

    public boolean deleteProject(String name) {
        ProjectDto project = getProject(name);
        if (project == null) {
            return true;
        }
        
        try {
            apiClient.deleteProject(project.getId());
            return true;
        } catch (Exception e) {
            log.error("Delete project failed: {}", e.getMessage());
            return false;
        }
    }

    public boolean deleteInventory(String name) {
        InventoryDto inventory = getInventory(name);
        if (inventory == null) {
            return true;
        }
        
        try {
            apiClient.deleteInventory(inventory.getId());
            return true;
        } catch (Exception e) {
            log.error("Delete inventory failed: {}", e.getMessage());
            return false;
        }
    }

    public boolean deleteJobTemplate(String name) {
        JobTemplateDto jobTemplate = getJobTemplate(name);
        if (jobTemplate == null) {
            return true;
        }
        
        try {
            apiClient.deleteJobTemplate(jobTemplate.getId());
            return true;
        } catch (Exception e) {
            log.error("Delete job template failed: {}", e.getMessage());
            return false;
        }
    }
} 