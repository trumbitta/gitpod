/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the MIT License. See License-MIT.txt in the project root for license information.
 */

{
  prometheusAlerts+:: {
      groups: [
        {
          name: "example",
          rules: [
            {
              alert: "HighRequestLatency",
              expr: "job:request_latency_seconds:mean5m{job=\"myjob\"} > 0.5",
              'for': "10m",
              labels: {
                severity: "critical"
              },
              annotations: {
                summary: "High request latency"
              }
            }
          ]
        }
      ]
},
}
