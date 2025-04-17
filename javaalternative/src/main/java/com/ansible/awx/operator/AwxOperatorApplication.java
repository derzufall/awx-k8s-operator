package com.ansible.awx.operator;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

/**
 * Main application class for the AWX Kubernetes Operator.
 */
@SpringBootApplication
@EnableScheduling
public class AwxOperatorApplication {

    public static void main(String[] args) {
        SpringApplication.run(AwxOperatorApplication.class, args);
    }
} 