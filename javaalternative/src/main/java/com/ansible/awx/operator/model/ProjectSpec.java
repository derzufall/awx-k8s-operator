package com.ansible.awx.operator.model;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Specification for an AWX Project.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProjectSpec {
    private String name;
    private String description;
    
    @Builder.Default
    private String scmType = "git";
    
    private String scmUrl;
    
    @Builder.Default
    private String scmBranch = "main";
    
    private String scmCredential;
} 