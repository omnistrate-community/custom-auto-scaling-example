# Omnistrate Internal API Guide for Custom Autoscaling

This guide explains how to interact with Omnistrate's internal sidecar API to implement custom autoscaling logic for your resources. While this repository provides a Go implementation, you can use **any programming language** that supports HTTP requests.

## Overview

When you deploy a service on Omnistrate with custom autoscaling enabled, a local sidecar API becomes available at `http://127.0.0.1:49750`. This API allows you to:

- Query the current capacity and status of a resource
- Add capacity units to a resource
- Remove capacity units from a resource

## API Endpoints

### Base URL

All API endpoints are accessible at:
```
http://127.0.0.1:49750/resource/{resourceAlias}
```

Where `{resourceAlias}` is the service key from your `omnistrate-compose.yaml` file.

### 1. Get Current Capacity

**Endpoint:** `GET /resource/{resourceAlias}/capacity`

**Description:** Retrieves the current capacity and status information for a resource.

**Response:**
```json
{
  "instanceId": "string",
  "resourceId": "string",
  "resourceAlias": "string",
  "status": "ACTIVE|STARTING|PAUSED|FAILED|UNKNOWN",
  "currentCapacity": 5,
  "lastObservedTimestamp": "2025-11-05T12:34:56.789Z"
}
```

**Status Values:**
- `ACTIVE` - Resource is running and ready
- `STARTING` - Resource is starting up
- `PAUSED` - Resource is paused
- `FAILED` - Resource has failed
- `UNKNOWN` - Status cannot be determined

### 2. Add Capacity

**Endpoint:** `POST /resource/{resourceAlias}/capacity/add`

**Description:** Adds capacity units to a resource.

**Request Body:**
```json
{
  "capacityToBeAdded": 2
}
```

**Response:**
```json
{
  "instanceId": "string",
  "resourceId": "string",
  "resourceAlias": "string"
}
```

### 3. Remove Capacity

**Endpoint:** `POST /resource/{resourceAlias}/capacity/remove`

**Description:** Removes capacity units from a resource.

**Request Body:**
```json
{
  "capacityToBeRemoved": 1
}
```

**Response:**
```json
{
  "instanceId": "string",
  "resourceId": "string",
  "resourceAlias": "string"
}
```

## Best Practices

### 1. Implement Cooldown Periods

Avoid rapid successive scaling operations by implementing a cooldown period (recommended: 5 minutes):

```
Last Scale Action → Wait 5 minutes → Next Scale Action
```

### 2. Wait for ACTIVE State

Always wait for a resource to reach `ACTIVE` state before performing the next scaling operation:

1. Check current status via GET capacity endpoint
2. If status is `STARTING`, poll until it becomes `ACTIVE`
3. If status is `FAILED`, handle the error appropriately
4. Only proceed with scaling when status is `ACTIVE`

### 3. Implement Step-Based Scaling

Scale gradually by adding/removing a fixed number of units per operation:

- Start with small steps (e.g., 1-2 units)
- Repeat operations until reaching target capacity
- Allow cooldown between steps

### 4. Use Retries with Exponential Backoff

Network requests can fail temporarily. Implement retry logic:

- Retry failed requests 3-5 times
- Use exponential backoff (1s, 2s, 4s, etc.)
- Set appropriate timeouts (e.g., 60 seconds per request)

### 5. Handle Errors Gracefully

- Check HTTP status codes (expect 200 for success)
- Parse error responses
- Log errors for debugging
- Implement fallback behavior

## Implementation Examples

### Go Implementation

This repository's reference implementation demonstrates best practices:

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

const baseURL = "http://127.0.0.1:49750/resource/"

type ResourceCapacity struct {
    InstanceID      string    `json:"instanceId"`
    ResourceID      string    `json:"resourceId"`
    ResourceAlias   string    `json:"resourceAlias"`
    Status          string    `json:"status"`
    CurrentCapacity int       `json:"currentCapacity"`
    LastObservedTS  time.Time `json:"lastObservedTimestamp"`
}

type ResourceInstance struct {
    InstanceID    string `json:"instanceId"`
    ResourceID    string `json:"resourceId"`
    ResourceAlias string `json:"resourceAlias"`
}

// GetCurrentCapacity retrieves current capacity information
func GetCurrentCapacity(ctx context.Context, resourceAlias string) (*ResourceCapacity, error) {
    url := fmt.Sprintf("%s%s/capacity", baseURL, resourceAlias)
    
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    client := &http.Client{Timeout: 60 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    var capacity ResourceCapacity
    if err := json.Unmarshal(body, &capacity); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    
    return &capacity, nil
}

// AddCapacity adds capacity units to a resource
func AddCapacity(ctx context.Context, resourceAlias string, capacityToAdd int) (*ResourceInstance, error) {
    url := fmt.Sprintf("%s%s/capacity/add", baseURL, resourceAlias)
    
    reqBody := map[string]int{"capacityToBeAdded": capacityToAdd}
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{Timeout: 60 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    var instance ResourceInstance
    if err := json.Unmarshal(body, &instance); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    
    return &instance, nil
}

// RemoveCapacity removes capacity units from a resource
func RemoveCapacity(ctx context.Context, resourceAlias string, capacityToRemove int) (*ResourceInstance, error) {
    url := fmt.Sprintf("%s%s/capacity/remove", baseURL, resourceAlias)
    
    reqBody := map[string]int{"capacityToBeRemoved": capacityToRemove}
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{Timeout: 60 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    var instance ResourceInstance
    if err := json.Unmarshal(body, &instance); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    
    return &instance, nil
}

// Example usage with proper waiting and cooldown
func ScaleToTarget(ctx context.Context, resourceAlias string, targetCapacity int) error {
    cooldownPeriod := 5 * time.Minute
    lastActionTime := time.Time{}
    
    for {
        // Enforce cooldown period
        if !lastActionTime.IsZero() && time.Since(lastActionTime) < cooldownPeriod {
            waitTime := cooldownPeriod - time.Since(lastActionTime)
            fmt.Printf("Waiting %v for cooldown period\n", waitTime)
            time.Sleep(waitTime)
        }
        
        // Get current capacity and status
        capacity, err := GetCurrentCapacity(ctx, resourceAlias)
        if err != nil {
            return fmt.Errorf("failed to get capacity: %w", err)
        }
        
        // Wait for ACTIVE state
        if capacity.Status != "ACTIVE" {
            if capacity.Status == "FAILED" {
                return fmt.Errorf("resource is in FAILED state")
            }
            fmt.Printf("Resource status: %s, waiting for ACTIVE...\n", capacity.Status)
            time.Sleep(30 * time.Second)
            continue
        }
        
        // Check if we've reached target
        if capacity.CurrentCapacity == targetCapacity {
            fmt.Println("Target capacity reached")
            break
        }
        
        // Scale up or down
        if capacity.CurrentCapacity < targetCapacity {
            fmt.Printf("Scaling up: %d -> %d\n", capacity.CurrentCapacity, capacity.CurrentCapacity+1)
            _, err = AddCapacity(ctx, resourceAlias, 1)
        } else {
            fmt.Printf("Scaling down: %d -> %d\n", capacity.CurrentCapacity, capacity.CurrentCapacity-1)
            _, err = RemoveCapacity(ctx, resourceAlias, 1)
        }
        
        if err != nil {
            return fmt.Errorf("failed to scale: %w", err)
        }
        
        lastActionTime = time.Now()
    }
    
    return nil
}
```

### Python Implementation

```python
import requests
import time
from typing import Dict, Optional
from datetime import datetime, timedelta

BASE_URL = "http://127.0.0.1:49750/resource"

class OmnistrateClient:
    def __init__(self, timeout: int = 60):
        self.timeout = timeout
        self.session = requests.Session()
    
    def get_current_capacity(self, resource_alias: str) -> Dict:
        """Get current capacity and status of a resource."""
        url = f"{BASE_URL}/{resource_alias}/capacity"
        
        try:
            response = self.session.get(url, timeout=self.timeout)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            raise Exception(f"Failed to get capacity: {e}")
    
    def add_capacity(self, resource_alias: str, capacity_to_add: int) -> Dict:
        """Add capacity units to a resource."""
        url = f"{BASE_URL}/{resource_alias}/capacity/add"
        payload = {"capacityToBeAdded": capacity_to_add}
        
        try:
            response = self.session.post(
                url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=self.timeout
            )
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            raise Exception(f"Failed to add capacity: {e}")
    
    def remove_capacity(self, resource_alias: str, capacity_to_remove: int) -> Dict:
        """Remove capacity units from a resource."""
        url = f"{BASE_URL}/{resource_alias}/capacity/remove"
        payload = {"capacityToBeRemoved": capacity_to_remove}
        
        try:
            response = self.session.post(
                url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=self.timeout
            )
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            raise Exception(f"Failed to remove capacity: {e}")
    
    def wait_for_active_state(
        self,
        resource_alias: str,
        max_wait_seconds: int = 900,
        check_interval: int = 30
    ) -> Dict:
        """Wait for resource to reach ACTIVE state."""
        start_time = datetime.now()
        
        while True:
            if (datetime.now() - start_time).total_seconds() > max_wait_seconds:
                raise TimeoutError("Timeout waiting for ACTIVE state")
            
            capacity = self.get_current_capacity(resource_alias)
            status = capacity.get("status")
            
            if status == "ACTIVE":
                return capacity
            elif status == "FAILED":
                raise Exception("Resource is in FAILED state")
            
            print(f"Resource status: {status}, waiting for ACTIVE...")
            time.sleep(check_interval)
    
    def scale_to_target(
        self,
        resource_alias: str,
        target_capacity: int,
        cooldown_seconds: int = 300,
        steps: int = 1
    ):
        """Scale resource to target capacity with cooldown and step-based scaling."""
        last_action_time = None
        
        while True:
            # Enforce cooldown period
            if last_action_time:
                elapsed = (datetime.now() - last_action_time).total_seconds()
                if elapsed < cooldown_seconds:
                    wait_time = cooldown_seconds - elapsed
                    print(f"Waiting {wait_time:.1f}s for cooldown period")
                    time.sleep(wait_time)
            
            # Wait for ACTIVE state and get current capacity
            capacity = self.wait_for_active_state(resource_alias)
            current = capacity["currentCapacity"]
            
            print(f"Current capacity: {current}, Target: {target_capacity}")
            
            # Check if target reached
            if current == target_capacity:
                print("Target capacity reached")
                break
            
            # Perform scaling operation
            try:
                if current < target_capacity:
                    scale_amount = min(steps, target_capacity - current)
                    print(f"Scaling up by {scale_amount} units")
                    self.add_capacity(resource_alias, scale_amount)
                else:
                    scale_amount = min(steps, current - target_capacity)
                    print(f"Scaling down by {scale_amount} units")
                    self.remove_capacity(resource_alias, scale_amount)
                
                last_action_time = datetime.now()
            
            except Exception as e:
                print(f"Scaling operation failed: {e}")
                raise

# Example usage
if __name__ == "__main__":
    client = OmnistrateClient()
    
    # Scale to target capacity
    try:
        client.scale_to_target(
            resource_alias="worker",
            target_capacity=5,
            cooldown_seconds=300,
            steps=1
        )
        print("Scaling completed successfully")
    except Exception as e:
        print(f"Scaling failed: {e}")
```

### Node.js (JavaScript) Implementation

```javascript
const axios = require('axios');

const BASE_URL = 'http://127.0.0.1:49750/resource';
const TIMEOUT = 60000; // 60 seconds

class OmnistrateClient {
  constructor(timeout = TIMEOUT) {
    this.timeout = timeout;
    this.client = axios.create({
      timeout: this.timeout,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  /**
   * Get current capacity and status of a resource
   */
  async getCurrentCapacity(resourceAlias) {
    try {
      const response = await this.client.get(
        `${BASE_URL}/${resourceAlias}/capacity`
      );
      return response.data;
    } catch (error) {
      throw new Error(`Failed to get capacity: ${error.message}`);
    }
  }

  /**
   * Add capacity units to a resource
   */
  async addCapacity(resourceAlias, capacityToAdd) {
    try {
      const response = await this.client.post(
        `${BASE_URL}/${resourceAlias}/capacity/add`,
        { capacityToBeAdded: capacityToAdd }
      );
      return response.data;
    } catch (error) {
      throw new Error(`Failed to add capacity: ${error.message}`);
    }
  }

  /**
   * Remove capacity units from a resource
   */
  async removeCapacity(resourceAlias, capacityToRemove) {
    try {
      const response = await this.client.post(
        `${BASE_URL}/${resourceAlias}/capacity/remove`,
        { capacityToBeRemoved: capacityToRemove }
      );
      return response.data;
    } catch (error) {
      throw new Error(`Failed to remove capacity: ${error.message}`);
    }
  }

  /**
   * Wait for resource to reach ACTIVE state
   */
  async waitForActiveState(
    resourceAlias,
    maxWaitSeconds = 900,
    checkInterval = 30
  ) {
    const startTime = Date.now();

    while (true) {
      if ((Date.now() - startTime) / 1000 > maxWaitSeconds) {
        throw new Error('Timeout waiting for ACTIVE state');
      }

      const capacity = await this.getCurrentCapacity(resourceAlias);
      const status = capacity.status;

      if (status === 'ACTIVE') {
        return capacity;
      } else if (status === 'FAILED') {
        throw new Error('Resource is in FAILED state');
      }

      console.log(`Resource status: ${status}, waiting for ACTIVE...`);
      await this.sleep(checkInterval * 1000);
    }
  }

  /**
   * Scale resource to target capacity with cooldown and step-based scaling
   */
  async scaleToTarget(
    resourceAlias,
    targetCapacity,
    cooldownSeconds = 300,
    steps = 1
  ) {
    let lastActionTime = null;

    while (true) {
      // Enforce cooldown period
      if (lastActionTime) {
        const elapsed = (Date.now() - lastActionTime) / 1000;
        if (elapsed < cooldownSeconds) {
          const waitTime = cooldownSeconds - elapsed;
          console.log(`Waiting ${waitTime.toFixed(1)}s for cooldown period`);
          await this.sleep(waitTime * 1000);
        }
      }

      // Wait for ACTIVE state and get current capacity
      const capacity = await this.waitForActiveState(resourceAlias);
      const current = capacity.currentCapacity;

      console.log(`Current capacity: ${current}, Target: ${targetCapacity}`);

      // Check if target reached
      if (current === targetCapacity) {
        console.log('Target capacity reached');
        break;
      }

      // Perform scaling operation
      try {
        if (current < targetCapacity) {
          const scaleAmount = Math.min(steps, targetCapacity - current);
          console.log(`Scaling up by ${scaleAmount} units`);
          await this.addCapacity(resourceAlias, scaleAmount);
        } else {
          const scaleAmount = Math.min(steps, current - targetCapacity);
          console.log(`Scaling down by ${scaleAmount} units`);
          await this.removeCapacity(resourceAlias, scaleAmount);
        }

        lastActionTime = Date.now();
      } catch (error) {
        console.error(`Scaling operation failed: ${error.message}`);
        throw error;
      }
    }
  }

  /**
   * Helper function to sleep
   */
  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Example usage
async function main() {
  const client = new OmnistrateClient();

  try {
    await client.scaleToTarget('worker', 5, 300, 1);
    console.log('Scaling completed successfully');
  } catch (error) {
    console.error(`Scaling failed: ${error.message}`);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = OmnistrateClient;
```

### Java Implementation

```java
import com.google.gson.Gson;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.time.Instant;

public class OmnistrateClient {
    private static final String BASE_URL = "http://127.0.0.1:49750/resource";
    private static final Duration TIMEOUT = Duration.ofSeconds(60);
    
    private final HttpClient httpClient;
    private final Gson gson;
    
    public OmnistrateClient() {
        this.httpClient = HttpClient.newBuilder()
            .connectTimeout(TIMEOUT)
            .build();
        this.gson = new Gson();
    }
    
    /**
     * Get current capacity and status of a resource
     */
    public ResourceCapacity getCurrentCapacity(String resourceAlias) throws Exception {
        String url = String.format("%s/%s/capacity", BASE_URL, resourceAlias);
        
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .timeout(TIMEOUT)
            .GET()
            .build();
        
        HttpResponse<String> response = httpClient.send(
            request,
            HttpResponse.BodyHandlers.ofString()
        );
        
        if (response.statusCode() != 200) {
            throw new Exception("Failed to get capacity, status: " + response.statusCode());
        }
        
        return gson.fromJson(response.body(), ResourceCapacity.class);
    }
    
    /**
     * Add capacity units to a resource
     */
    public ResourceInstance addCapacity(String resourceAlias, int capacityToAdd) throws Exception {
        String url = String.format("%s/%s/capacity/add", BASE_URL, resourceAlias);
        
        JsonObject payload = new JsonObject();
        payload.addProperty("capacityToBeAdded", capacityToAdd);
        
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .timeout(TIMEOUT)
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(gson.toJson(payload)))
            .build();
        
        HttpResponse<String> response = httpClient.send(
            request,
            HttpResponse.BodyHandlers.ofString()
        );
        
        if (response.statusCode() != 200) {
            throw new Exception("Failed to add capacity, status: " + response.statusCode());
        }
        
        return gson.fromJson(response.body(), ResourceInstance.class);
    }
    
    /**
     * Remove capacity units from a resource
     */
    public ResourceInstance removeCapacity(String resourceAlias, int capacityToRemove) throws Exception {
        String url = String.format("%s/%s/capacity/remove", BASE_URL, resourceAlias);
        
        JsonObject payload = new JsonObject();
        payload.addProperty("capacityToBeRemoved", capacityToRemove);
        
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .timeout(TIMEOUT)
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(gson.toJson(payload)))
            .build();
        
        HttpResponse<String> response = httpClient.send(
            request,
            HttpResponse.BodyHandlers.ofString()
        );
        
        if (response.statusCode() != 200) {
            throw new Exception("Failed to remove capacity, status: " + response.statusCode());
        }
        
        return gson.fromJson(response.body(), ResourceInstance.class);
    }
    
    /**
     * Wait for resource to reach ACTIVE state
     */
    public ResourceCapacity waitForActiveState(
        String resourceAlias,
        int maxWaitSeconds,
        int checkInterval
    ) throws Exception {
        Instant startTime = Instant.now();
        
        while (true) {
            if (Duration.between(startTime, Instant.now()).getSeconds() > maxWaitSeconds) {
                throw new Exception("Timeout waiting for ACTIVE state");
            }
            
            ResourceCapacity capacity = getCurrentCapacity(resourceAlias);
            String status = capacity.status;
            
            if ("ACTIVE".equals(status)) {
                return capacity;
            } else if ("FAILED".equals(status)) {
                throw new Exception("Resource is in FAILED state");
            }
            
            System.out.println("Resource status: " + status + ", waiting for ACTIVE...");
            Thread.sleep(checkInterval * 1000L);
        }
    }
    
    /**
     * Scale resource to target capacity with cooldown and step-based scaling
     */
    public void scaleToTarget(
        String resourceAlias,
        int targetCapacity,
        int cooldownSeconds,
        int steps
    ) throws Exception {
        Instant lastActionTime = null;
        
        while (true) {
            // Enforce cooldown period
            if (lastActionTime != null) {
                long elapsed = Duration.between(lastActionTime, Instant.now()).getSeconds();
                if (elapsed < cooldownSeconds) {
                    long waitTime = cooldownSeconds - elapsed;
                    System.out.println("Waiting " + waitTime + "s for cooldown period");
                    Thread.sleep(waitTime * 1000);
                }
            }
            
            // Wait for ACTIVE state and get current capacity
            ResourceCapacity capacity = waitForActiveState(resourceAlias, 900, 30);
            int current = capacity.currentCapacity;
            
            System.out.println("Current capacity: " + current + ", Target: " + targetCapacity);
            
            // Check if target reached
            if (current == targetCapacity) {
                System.out.println("Target capacity reached");
                break;
            }
            
            // Perform scaling operation
            if (current < targetCapacity) {
                int scaleAmount = Math.min(steps, targetCapacity - current);
                System.out.println("Scaling up by " + scaleAmount + " units");
                addCapacity(resourceAlias, scaleAmount);
            } else {
                int scaleAmount = Math.min(steps, current - targetCapacity);
                System.out.println("Scaling down by " + scaleAmount + " units");
                removeCapacity(resourceAlias, scaleAmount);
            }
            
            lastActionTime = Instant.now();
        }
    }
    
    // Data classes
    public static class ResourceCapacity {
        public String instanceId;
        public String resourceId;
        public String resourceAlias;
        public String status;
        public int currentCapacity;
        public String lastObservedTimestamp;
    }
    
    public static class ResourceInstance {
        public String instanceId;
        public String resourceId;
        public String resourceAlias;
    }
    
    // Example usage
    public static void main(String[] args) {
        OmnistrateClient client = new OmnistrateClient();
        
        try {
            client.scaleToTarget("worker", 5, 300, 1);
            System.out.println("Scaling completed successfully");
        } catch (Exception e) {
            System.err.println("Scaling failed: " + e.getMessage());
            System.exit(1);
        }
    }
}
```

## Common Patterns

### Pattern 1: Simple Scale Up/Down

For simple use cases where you want to manually trigger scaling:

```python
# Scale up by 2 units
client.add_capacity("worker", 2)

# Scale down by 1 unit
client.remove_capacity("worker", 1)
```

### Pattern 2: Metric-Based Scaling

Scale based on custom metrics (CPU, memory, queue length, etc.):

```python
def scale_based_on_queue_length(client, resource_alias):
    queue_length = get_queue_length()  # Your custom metric
    capacity = client.get_current_capacity(resource_alias)
    current = capacity["currentCapacity"]
    
    # Scale up if queue > 100 items per unit
    if queue_length > current * 100:
        target = (queue_length // 100) + 1
        client.scale_to_target(resource_alias, target)
    
    # Scale down if queue < 50 items per unit
    elif queue_length < current * 50 and current > 1:
        target = max(1, queue_length // 50)
        client.scale_to_target(resource_alias, target)
```

### Pattern 3: Scheduled Scaling

Scale based on time of day or day of week:

```python
from datetime import datetime

def scale_based_on_schedule(client, resource_alias):
    hour = datetime.now().hour
    
    # High traffic hours: 9 AM - 5 PM
    if 9 <= hour < 17:
        target_capacity = 10
    # Low traffic hours
    else:
        target_capacity = 2
    
    client.scale_to_target(resource_alias, target_capacity)
```

### Pattern 4: Predictive Scaling

Scale based on predicted load:

```python
def predictive_scaling(client, resource_alias):
    predicted_load = get_load_prediction()  # ML model prediction
    
    # Calculate required capacity
    target_capacity = max(1, int(predicted_load / 100))
    
    # Apply scaling with buffer
    client.scale_to_target(resource_alias, target_capacity)
```

## Troubleshooting

### Issue: Connection Refused

**Symptom:** Cannot connect to `http://127.0.0.1:49750`

**Solution:** The sidecar API is only available when running on Omnistrate. Ensure your controller is deployed as part of an Omnistrate service, not running locally.

### Issue: Timeout Errors

**Symptom:** Requests timeout frequently

**Solution:**
- Increase timeout values
- Check network connectivity
- Verify resource is not stuck in a transitional state

### Issue: Rapid Scaling

**Symptom:** Resource scales up and down rapidly

**Solution:**
- Increase cooldown period
- Implement hysteresis (different thresholds for scale up vs scale down)
- Add dampening to your scaling logic

## Additional Resources

- [Omnistrate Documentation](https://docs.omnistrate.com)
- [Custom Auto Scaling Example Repository](https://github.com/omnistrate-community/custom-auto-scaling-example)
- [Omnistrate Community Examples](https://github.com/omnistrate-community)
