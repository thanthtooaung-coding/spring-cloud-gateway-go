# Spring Cloud Gateway with Go Microservices

This project demonstrates a polyglot microservice architecture featuring Java-based Spring Cloud components acting as the control plane for backend microservices written in Go. It showcases the use of an API Gateway for routing and a Discovery Service for dynamic service registration and discovery.

## Architecture

The system is composed of four main services that work together:

1.  **Discovery Service (Eureka):** A service registry where all other services register themselves. This allows services to find each other by name without needing to know their hardcoded IP addresses or ports.
2.  **Go Microservices (Product & Order APIs):** Two independent backend services written in Go. They handle specific business logic (managing products and orders) and register themselves with the Discovery Service upon startup.
3.  **API Gateway:** The single entry point for all external clients. It routes incoming requests to the appropriate Go microservice by looking up their location in the Discovery Service.

<!-- end list -->

```
+-----------+      +-------------------------+      +--------------------+
|           |      |      API Gateway        |      |  Go Product API    |
|  Client   | ---->|  (Spring, port 8080)    | ---->|  (Go, port 9091)   |
| (cURL/UI) |      |                         |      |                    |
+-----------+      +-------------------------+      +--------------------+
                              |                            ^
                              |                            |
                              |              +-------------------------+
                              |              |                         |
                              +------------->| Discovery Service       |
                                             |(Spring Eureka,port:8761)|
                                             |                         |
                              +------------->|                         |
                              |              +-------------------------+
                              |                            ^
                              |                            |
                              |              +--------------------+
                              |              |   Go Order API     |
                              +------------->|   (Go, port 9092)  |
                                             |                    |
                                             +--------------------+
```

## Technologies Used

  * **Java / Spring:**
      * Spring Boot 3
      * Spring Cloud Gateway
      * Spring Cloud Netflix Eureka (Server & Client)
      * Maven
  * **Go:**
      * Standard Library (`net/http`)
      * Go Modules for dependency management
      * `hudl/fargo` library for Eureka client registration
  * **Platform:**
      * Java 17+
      * Go 1.18+

## Project Structure

Your screenshot shows a well-organized multi-module project:

```
.
├── api-gateway/            # The Spring Cloud Gateway application
├── discovery-service/      # The Spring Cloud Eureka Server application
├── order-api/              # The Go microservice for orders (CRUD)
├── product-api/            # The Go microservice for products
├── mvnw                    # Maven wrapper for building Java modules
└── pom.xml                 # Root Maven POM file for the project
```

## Prerequisites

Before you begin, ensure you have the following installed:

  * **Java Development Kit (JDK)**: Version 17 or higher.
  * **Apache Maven**: To build the Java projects.
  * **Go**: Version 1.18 or higher.

## How to Run the System

To run the entire application stack, you must follow these steps in order.

### 1\. Build the Java Modules

From the root directory (`SPRING-CLOUD-GATEWAY-GO`), run the Maven wrapper to build the `discovery-service` and `api-gateway` JAR files.

```bash
./mvnw clean install
```

### 2\. Prepare the Go Modules

Navigate into each Go directory and run `go mod tidy` to ensure the dependencies (like the Fargo Eureka client) are downloaded.

```bash
# For the product API
cd product-api
go mod tidy
cd ..

# For the order API
cd order-api
go mod tidy
cd ..
```

### 3\. Run the Applications (In Order)

You will need **four separate terminal windows** to run all the services.

**a. Start the Discovery Service (Eureka Server)**
This must be the first service to start.

```bash
# In Terminal 1
java -jar discovery-service/target/discovery-service-1.0.0-SNAPSHOT.jar
```

Wait for it to start. You can view the Eureka dashboard by navigating to **`http://localhost:8761`** in your browser.

**b. Start the Go Microservices**
These can be started in any order after the discovery service is up.

```bash
# In Terminal 2
cd product-api
go run main.go

# In Terminal 3
cd order-api
go run main.go
```

Watch their logs for the "Successfully registered with Eureka..." message. Refresh the Eureka dashboard to see `PRODUCT-SERVICE` and `ORDER-SERVICE` appear.

**c. Start the API Gateway**
This is the final piece. It will register with Eureka and begin routing requests.

```bash
# In Terminal 4
java -jar api-gateway/target/api-gateway-1.0.0-SNAPSHOT.jar
```

## Testing the API Endpoints

All requests should be sent to the API Gateway on port `8080`.

### Product Service

| Method | Path                  | Description           |
| :---   | :---                  | :---                  |
| `GET`  | `/api/products`       | Retrieves all products. |

**Example `curl` command:**

```bash
curl http://localhost:8080/api/products
```

### Order Service (CRUD)

| Method   | Path                  | Description                |
| :---     | :---                  | :---                       |
| `POST`   | `/api/orders`         | Creates a new order.       |
| `GET`    | `/api/orders/{id}`    | Retrieves a single order.  |
| `PUT`    | `/api/orders/{id}`    | Updates an existing order. |
| `DELETE` | `/api/orders/{id}`    | Deletes an order.          |

**Example `curl` commands:**

```bash
# CREATE a new order
curl -X POST -H "Content-Type: application/json" -d '{"product": "Keyboard", "total": 75.99}' http://localhost:8080/api/orders

# GET all orders
curl http://localhost:8080/api/orders

# GET a single order by its ID
curl http://localhost:8080/api/orders/o101

# UPDATE an existing order
curl -X PUT -H "Content-Type: application/json" -d '{"product": "Wireless Gaming Mouse", "total": 45.50}' http://localhost:8080/api/orders/o102

# DELETE an order
curl -X DELETE http://localhost:8080/api/orders/o101
```

## Configuration

  * **Java Services**: Configuration is located in the `src/main/resources/application.properties` file of each module (`api-gateway`, `discovery-service`).
  * **Go Services**: The Eureka server URL is currently hardcoded in the `main.go` file of each module. In a real-world scenario, this would be externalized.

The key configuration is the routing rules in the **API Gateway**, which dynamically forward requests to the correct service based on its name in Eureka.

```properties
# Example from api-gateway/application.properties
# This rule forwards requests from /api/orders/** to the ORDER-SERVICE

spring.cloud.gateway.routes[1].id=order-service-route
spring.cloud.gateway.routes[1].uri=lb://ORDER-SERVICE
spring.cloud.gateway.routes[1].predicates[0]=Path=/api/orders/**
spring.cloud.gateway.routes[1].filters[0]=StripPrefix=1
```
