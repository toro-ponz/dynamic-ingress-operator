# Dynamic Ingress Operator

A operator that change ingress spec dynamically by external api (or k8s resource).

## CRD

### DynamicIngress

```yaml
apiVersion: ingress.toroponz.io/v1beta1
kind: DynamicIngress
metadata:
  name: dynamic-ingress-sample
spec:
  target:
    name: test
    namespace: dynamic-ingress-operator-system
  passiveIngress: null
  activeIngress:
    template:
      metadata:
        annotations:
          alb.ingress.kubernetes.io/group.name: test
          alb.ingress.kubernetes.io/scheme: internet-facing
          alb.ingress.kubernetes.io/target-type: ip
          kubernetes.io/ingress.class: alb
      spec:
        rules:
          - http:
              paths:
                - backend:
                    service:
                      name: test-service
                      port:
                        number: 80
                  pathType: ImplementationSpecific
  state: dynamic-ingress-state-sample
  expected:
    status: '200'
    body: '{"status":"maintenance"}'
    policy: contains
  errorPolicy: retain
```

### DynamicIngressState

#### Fixed Mode

```yaml
apiVersion: ingress.toroponz.io/v1beta1
kind: DynamicIngressState
metadata:
  name: dynamic-ingress-state-sample
spec:
  fixedResponse:
    status: '200'
    body: |
      {
        "status": "ok"
      }
```

#### Probe Mode

A mode for calling external HTTP APIs per interval.

```yaml
apiVersion: ingress.toroponz.io/v1beta1
kind: DynamicIngressState
metadata:
  name: dynamic-ingress-state-sample
spec:
  probe:
    type: HTTP
    method: GET
    url: https://api.toroponz.io/status
    interval: 60s
```
