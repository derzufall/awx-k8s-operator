package com.ansible.awx.operator.model;

import lombok.Data;
import java.util.List;

/**
 * Specification for an AWX Inventory.
 */
@Data
public class InventorySpec {
    private String name;
    private String description;
    private String variables;
    private List<HostSpec> hosts;
} 