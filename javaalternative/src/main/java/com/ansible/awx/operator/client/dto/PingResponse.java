package com.ansible.awx.operator.client.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Data Transfer Object for AWX ping response
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonIgnoreProperties(ignoreUnknown = true)
public class PingResponse {
    @JsonProperty("instance_group_status")
    private String instanceGroupStatus;
    
    private String version;
    private Boolean ha;
    private Boolean online;
} 