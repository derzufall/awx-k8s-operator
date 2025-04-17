package com.ansible.awx.operator.config;

import com.ansible.awx.operator.controller.AWXInstanceReconciler;
import com.ansible.awx.operator.model.AWXInstance;
import io.kubernetes.client.extended.controller.Controller;
import io.kubernetes.client.extended.controller.ControllerManager;
import io.kubernetes.client.extended.controller.builder.ControllerBuilder;
import io.kubernetes.client.extended.controller.builder.DefaultControllerBuilder;
import io.kubernetes.client.informer.SharedIndexInformer;
import io.kubernetes.client.informer.SharedInformerFactory;
import io.kubernetes.client.openapi.ApiClient;
import io.kubernetes.client.openapi.apis.CustomObjectsApi;
import io.kubernetes.client.util.generic.GenericKubernetesApi;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.time.Duration;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Configuration for the Kubernetes controller.
 */
@Configuration
public class ControllerConfig {
    private static final String GROUP = "awx.ansible.com";
    private static final String VERSION = "v1alpha1";
    private static final String PLURAL = "awxinstances";
    private static final String KIND = "AWXInstance";

    @Bean
    public ExecutorService executorService() {
        return Executors.newFixedThreadPool(10);
    }

    @Bean
    public CustomObjectsApi customObjectsApi(ApiClient apiClient) {
        return new CustomObjectsApi(apiClient);
    }

    @Bean
    public GenericKubernetesApi<AWXInstance, AWXInstance> awxInstanceApi(ApiClient apiClient) {
        return new GenericKubernetesApi<>(
                AWXInstance.class,
                AWXInstance.class,
                GROUP,
                VERSION,
                PLURAL,
                apiClient);
    }

    @Bean
    public SharedIndexInformer<AWXInstance> awxInstanceInformer(
            SharedInformerFactory informerFactory,
            GenericKubernetesApi<AWXInstance, AWXInstance> awxInstanceApi) {
        
        return informerFactory.sharedIndexInformerFor(
                awxInstanceApi,
                AWXInstance.class,
                0);
    }

    @Bean
    public Controller awxInstanceController(
            SharedInformerFactory informerFactory,
            AWXInstanceReconciler reconciler,
            SharedIndexInformer<AWXInstance> awxInstanceInformer) {
        
        return ControllerBuilder.defaultBuilder(informerFactory)
                .watch(workQueue -> DefaultControllerBuilder.controllerWatchBuilder(AWXInstance.class, workQueue)
                        .withResyncPeriod(Duration.ofMinutes(1))
                        .build())
                .withWorkerCount(2)
                .withReadyFunc(awxInstanceInformer::hasSynced)
                .withReconciler(reconciler)
                .withName("AWXInstanceController")
                .build();
    }

    @Bean
    public ControllerManager controllerManager(
            SharedInformerFactory informerFactory,
            Controller awxInstanceController,
            ExecutorService executorService) {
        
        ControllerManager manager = new ControllerManager(informerFactory, awxInstanceController);
        manager.setExecutorService(executorService);
        return manager;
    }
} 