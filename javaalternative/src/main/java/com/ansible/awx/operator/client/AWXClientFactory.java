package com.ansible.awx.operator.client;

import com.ansible.awx.operator.model.AWXConnectionConfig;
import org.springframework.http.HttpHeaders;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.client.WebClient;
import org.springframework.web.reactive.function.client.support.WebClientAdapter;
import org.springframework.web.service.invoker.HttpServiceProxyFactory;

import java.nio.charset.StandardCharsets;
import java.util.Base64;

/**
 * Factory for creating AWXClient instances.
 */
@Component
public class AWXClientFactory {

    /**
     * Creates an AWXClient instance for the given connection configuration.
     *
     * @param config The AWX connection configuration
     * @return A configured AWXClient
     */
    public AWXClient createClient(AWXConnectionConfig config) {
        WebClient webClient = WebClient.builder()
                .baseUrl(config.getUrl())
                .defaultHeader(HttpHeaders.AUTHORIZATION, createBasicAuthHeader(config.getUsername(), config.getPassword()))
                .build();
        
        WebClientAdapter adapter = WebClientAdapter.forClient(webClient);
        HttpServiceProxyFactory factory = HttpServiceProxyFactory.builderFor(adapter).build();
        AWXApiClient apiClient = factory.createClient(AWXApiClient.class);
        
        return new AWXClient(apiClient);
    }

    private String createBasicAuthHeader(String username, String password) {
        String credentials = username + ":" + password;
        byte[] encodedCredentials = Base64.getEncoder().encode(credentials.getBytes(StandardCharsets.UTF_8));
        return "Basic " + new String(encodedCredentials);
    }
} 