package com.ansible.awx.operator.client.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Data Transfer Object for AWX Project
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonIgnoreProperties(ignoreUnknown = true)
public class ProjectDto {
    private Integer id;
    private String name;
    private String description;
    
    @JsonProperty("scm_type")
    private String scmType;
    
    @JsonProperty("scm_url")
    private String scmUrl;
    
    @JsonProperty("scm_branch")
    private String scmBranch;
    
    @JsonProperty("scm_credential")
    private Integer scmCredential;
    
    @JsonProperty("local_path")
    private String localPath;
    
    private String status;
} 