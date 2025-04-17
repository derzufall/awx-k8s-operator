package com.ansible.awx.operator.model;

import lombok.Data;
import java.util.List;

/**
 * Specification for an AWXInstance custom resource.
 */
@Data
public class AWXInstanceSpec {
    private String adminUser;
    private String adminPassword;
    private String adminEmail;
    private String hostname;
    private String protocol = "https";
    private boolean externalInstance;
    private int replicas = 1;
    private List<ProjectSpec> projects;
    private List<InventorySpec> inventories;
    private List<JobTemplateSpec> jobTemplates;
} 