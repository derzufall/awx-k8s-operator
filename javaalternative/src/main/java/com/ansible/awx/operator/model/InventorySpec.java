package com.ansible.awx.operator.model;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import java.util.List;

/**
 * Specification for an AWX Inventory.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class InventorySpec {
    private String name;
    private String description;
    private String variables;
    private List<HostSpec> hosts;
} 