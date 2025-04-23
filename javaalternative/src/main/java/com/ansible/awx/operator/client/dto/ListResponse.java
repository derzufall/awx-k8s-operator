package com.ansible.awx.operator.client.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

/**
 * Generic Data Transfer Object for AWX paginated list responses
 * @param <T> The type of item in the results list
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
@JsonIgnoreProperties(ignoreUnknown = true)
public class ListResponse<T> {
    private Integer count;
    private String next;
    private String previous;
    private List<T> results;
} 