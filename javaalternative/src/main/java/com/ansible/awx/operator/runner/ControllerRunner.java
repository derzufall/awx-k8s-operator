package com.ansible.awx.operator.runner;

import io.kubernetes.client.extended.controller.ControllerManager;
import io.kubernetes.client.informer.SharedInformerFactory;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.CommandLineRunner;
import org.springframework.stereotype.Component;

/**
 * Runner to start the controller manager.
 */
@Component
@Slf4j
public class ControllerRunner implements CommandLineRunner {
    private final SharedInformerFactory informerFactory;
    private final ControllerManager controllerManager;

    @Autowired
    public ControllerRunner(
            SharedInformerFactory informerFactory,
            ControllerManager controllerManager) {
        this.informerFactory = informerFactory;
        this.controllerManager = controllerManager;
    }

    @Override
    public void run(String... args) throws Exception {
        log.info("Starting informer factory and controller manager");
        
        informerFactory.startAllRegisteredInformers();
        controllerManager.run();
    }
} 