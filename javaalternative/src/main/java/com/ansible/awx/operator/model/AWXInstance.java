package com.ansible.awx.operator.model;

import io.kubernetes.client.common.KubernetesObject;
import io.kubernetes.client.openapi.models.V1ObjectMeta;
import lombok.Data;

/**
 * Represents an AWXInstance custom resource.
 */
@Data
public class AWXInstance implements KubernetesObject {
    private String apiVersion;
    private String kind;
    private V1ObjectMeta metadata;
    private AWXInstanceSpec spec;
    private AWXInstanceStatus status;
} 