package com.ansible.awx.operator.model;

import io.kubernetes.client.openapi.models.V1Condition;
import lombok.Data;
import java.time.OffsetDateTime;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Status for an AWXInstance custom resource.
 */
@Data
public class AWXInstanceStatus {
    private List<V1Condition> conditions = new ArrayList<>();
    private Map<String, String> projectStatuses = new HashMap<>();
    private Map<String, String> inventoryStatuses = new HashMap<>();
    private Map<String, String> jobTemplateStatuses = new HashMap<>();
    private OffsetDateTime lastConnectionCheck;
    private String connectionStatus;
} 