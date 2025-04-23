package com.ansible.awx.operator.model;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import java.util.List;

/**
 * Specification for an AWX instance custom resource.
 * This would be the top-level CR that contains connection information.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class AWXInstanceSpec {
    private String name;
    private AWXConnectionConfig connection;
    
    // Optional resources to manage
    private List<ProjectSpec> projects;
    private List<InventorySpec> inventories;
    private List<JobTemplateSpec> jobTemplates;
} 