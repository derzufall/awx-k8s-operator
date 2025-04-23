package com.ansible.awx.operator.model;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Configuration for AWX connection derived from CRD.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AWXConnectionConfig {
    private String url;
    private String username;
    private String password;
    
    @Builder.Default
    private Boolean validateCerts = true;
    
    @Builder.Default
    private Integer timeout = 60;
} 