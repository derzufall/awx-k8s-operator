package com.ansible.awx.operator.model;

import lombok.Data;

/**
 * Specification for an AWX Job Template.
 */
@Data
public class JobTemplateSpec {
    private String name;
    private String description;
    private String projectName;
    private String inventoryName;
    private String playbook;
    private String extraVars;
} 