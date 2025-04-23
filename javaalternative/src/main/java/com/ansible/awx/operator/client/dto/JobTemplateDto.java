package com.ansible.awx.operator.client.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Data Transfer Object for AWX Job Template
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonIgnoreProperties(ignoreUnknown = true)
public class JobTemplateDto {
    private Integer id;
    private String name;
    private String description;
    private Integer project;
    private Integer inventory;
    private String playbook;
    
    @JsonProperty("extra_vars")
    private String extraVars;
    
    @JsonProperty("job_type")
    @Builder.Default
    private String jobType = "run";
    
    @JsonProperty("ask_limit_on_launch")
    @Builder.Default
    private Boolean askLimitOnLaunch = false;
    
    @JsonProperty("ask_variables_on_launch")
    @Builder.Default
    private Boolean askVariablesOnLaunch = false;
    
    @JsonProperty("ask_inventory_on_launch")
    @Builder.Default
    private Boolean askInventoryOnLaunch = false;
    
    @JsonProperty("ask_credential_on_launch")
    @Builder.Default
    private Boolean askCredentialOnLaunch = false;
} 