package com.ansible.awx.operator.client;

import com.ansible.awx.operator.client.dto.InventoryDto;
import com.ansible.awx.operator.client.dto.JobTemplateDto;
import com.ansible.awx.operator.client.dto.ListResponse;
import com.ansible.awx.operator.client.dto.PingResponse;
import com.ansible.awx.operator.client.dto.ProjectDto;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.service.annotation.DeleteExchange;
import org.springframework.web.service.annotation.GetExchange;
import org.springframework.web.service.annotation.HttpExchange;
import org.springframework.web.service.annotation.PostExchange;
import org.springframework.web.service.annotation.PutExchange;

/**
 * HttpInterface for AWX API
 */
@HttpExchange(accept = MediaType.APPLICATION_JSON_VALUE)
public interface AWXApiClient {

    /**
     * Ping AWX to test connectivity
     */
    @GetExchange("/api/v2/ping/")
    PingResponse ping();

    /**
     * Get projects with optional filtering
     */
    @GetExchange("/api/v2/projects/?name={name}")
    ListResponse<ProjectDto> getProjects(@PathVariable("name") String name);

    /**
     * Create a new project
     */
    @PostExchange(value = "/api/v2/projects/", contentType = MediaType.APPLICATION_JSON_VALUE)
    ProjectDto createProject(@RequestBody ProjectDto projectDto);

    /**
     * Update an existing project
     */
    @PutExchange(value = "/api/v2/projects/{id}/", contentType = MediaType.APPLICATION_JSON_VALUE)
    ProjectDto updateProject(@PathVariable("id") Integer id, @RequestBody ProjectDto projectDto);

    /**
     * Delete a project
     */
    @DeleteExchange("/api/v2/projects/{id}/")
    void deleteProject(@PathVariable("id") Integer id);

    /**
     * Get inventories with optional filtering
     */
    @GetExchange("/api/v2/inventories/?name={name}")
    ListResponse<InventoryDto> getInventories(@PathVariable("name") String name);

    /**
     * Create a new inventory
     */
    @PostExchange(value = "/api/v2/inventories/", contentType = MediaType.APPLICATION_JSON_VALUE)
    InventoryDto createInventory(@RequestBody InventoryDto inventoryDto);

    /**
     * Update an existing inventory
     */
    @PutExchange(value = "/api/v2/inventories/{id}/", contentType = MediaType.APPLICATION_JSON_VALUE)
    InventoryDto updateInventory(@PathVariable("id") Integer id, @RequestBody InventoryDto inventoryDto);

    /**
     * Delete an inventory
     */
    @DeleteExchange("/api/v2/inventories/{id}/")
    void deleteInventory(@PathVariable("id") Integer id);

    /**
     * Get job templates with optional filtering
     */
    @GetExchange("/api/v2/job_templates/?name={name}")
    ListResponse<JobTemplateDto> getJobTemplates(@PathVariable("name") String name);

    /**
     * Create a new job template
     */
    @PostExchange(value = "/api/v2/job_templates/", contentType = MediaType.APPLICATION_JSON_VALUE)
    JobTemplateDto createJobTemplate(@RequestBody JobTemplateDto jobTemplateDto);

    /**
     * Update an existing job template
     */
    @PutExchange(value = "/api/v2/job_templates/{id}/", contentType = MediaType.APPLICATION_JSON_VALUE)
    JobTemplateDto updateJobTemplate(@PathVariable("id") Integer id, @RequestBody JobTemplateDto jobTemplateDto);

    /**
     * Delete a job template
     */
    @DeleteExchange("/api/v2/job_templates/{id}/")
    void deleteJobTemplate(@PathVariable("id") Integer id);
} 