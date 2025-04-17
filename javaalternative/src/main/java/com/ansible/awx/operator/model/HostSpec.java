package com.ansible.awx.operator.model;

import lombok.Data;

/**
 * Specification for an AWX Inventory Host.
 */
@Data
public class HostSpec {
    private String name;
    private String description;
    private String variables;
} 