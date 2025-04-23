package com.ansible.awx.operator.client.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Data Transfer Object for AWX Inventory
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonIgnoreProperties(ignoreUnknown = true)
public class InventoryDto {
    private Integer id;
    private String name;
    private String description;
    private String variables;
    private Integer organization;
    private String kind;
    
    @JsonProperty("host_filter")
    private String hostFilter;
    
    @JsonProperty("has_active_failures")
    private Boolean hasActiveFailures;
    
    @JsonProperty("total_hosts")
    private Integer totalHosts;
    
    @JsonProperty("hosts_with_active_failures")
    private Integer hostsWithActiveFailures;
    
    @JsonProperty("total_groups")
    private Integer totalGroups;
    
    @JsonProperty("has_inventory_sources")
    private Boolean hasInventorySources;
    
    @JsonProperty("total_inventory_sources")
    private Integer totalInventorySources;
    
    @JsonProperty("inventory_sources_with_failures")
    private Integer inventorySourcesWithFailures;
    
    private String created;
    private String modified;
} 