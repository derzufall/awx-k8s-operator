# Build stage
FROM maven:3.9.6-eclipse-temurin-17 AS build
WORKDIR /workspace/app

# Copy the Maven POM and source code
COPY pom.xml .
COPY src ./src

# Build the application
RUN mvn clean package -DskipTests

# Runtime stage
FROM eclipse-temurin:17-jre
WORKDIR /app

# Copy the built artifact from the build stage
COPY --from=build /workspace/app/target/*.jar app.jar

# Set the entrypoint
ENTRYPOINT ["java", "-jar", "app.jar"] 