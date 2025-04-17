package com.ansible.awx.operator.config;

import io.kubernetes.client.openapi.ApiClient;
import io.kubernetes.client.util.ClientBuilder;
import io.kubernetes.client.util.KubeConfig;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.FileReader;
import java.io.IOException;

/**
 * Configuration for the Kubernetes client.
 */
@Configuration
public class KubernetesClientConfig {

    @Bean
    public ApiClient kubernetesApiClient() throws IOException {
        // Try to load from service account token first (when running in cluster)
        try {
            ApiClient client = ClientBuilder.cluster().build();
            // Increase timeout for long-running operations
            client.setReadTimeout(60000);
            return client;
        } catch (IOException e) {
            // Fallback to kubeconfig file for local development
            String kubeConfigPath = System.getProperty("user.home") + "/.kube/config";
            KubeConfig kubeConfig = KubeConfig.loadKubeConfig(new FileReader(kubeConfigPath));
            ApiClient client = ClientBuilder.kubeconfig(kubeConfig).build();
            client.setReadTimeout(60000);
            return client;
        }
    }
} 