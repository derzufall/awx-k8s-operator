package com.ansible.awx.operator.model;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Specification for an AWX Job Template.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class JobTemplateSpec {
    private String name;
    private String description;
    private String projectName;
    private String inventoryName;
    private String playbook;
    private String extraVars;
} 