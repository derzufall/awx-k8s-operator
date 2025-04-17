package com.ansible.awx.operator.model;

import lombok.Data;

/**
 * Specification for an AWX Project.
 */
@Data
public class ProjectSpec {
    private String name;
    private String description;
    private String scmType = "git";
    private String scmUrl;
    private String scmBranch = "main";
    private String scmCredential;
} 