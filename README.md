# Model Data Collector Dev Box

## Setup

### 1. Pull images

```
    az acr login -n vmagentacr
    docker pull vmagentacr.azurecr.io/mdc/xds:latest
    docker pull vmagentacr.azurecr.io/public/mir/mir-mdc:55585752.1643013235733
    docker pull envoyproxy/envoy:v1.16.1
```

### 2. Download cert for XDS
```
    az keyvault secret download --name cert-mirtest-v2-test-endpt-20201117173353-1ef841f5-52d7-4b45-bb1f-c080c67d6fe5 --vault-name infer-dep-usw2dev --file xds/server_cert.pem
```

### 3. Setup Auth for eventhub

    Please create a .env file with following content:
```
    eventhubConnStr=Endpoint={connect string to eventhub instance}  
    blobName=<storage account name>
    blobKey=<stroage account key >
```
    Note: The connection string of EventHub Instance `EntityPath`  (not the eventhub namespace str)

### 4. Using your own model
open `docker-compose.yaml`, replace the `model` part to use your own model
```
model:
    image: mendhak/http-https-echo:23
    environment:
        - HTTP_PORT=5001
    ports:
        - "5001:5001"
```

### 5. Test it
boot up the stack
```
    docker-compose up -d
```


Access your model server at port 10001(envoy)  
```
    curl -X PUT http://localhost:10001/score      
```

You should see logs of mdc like following
```
docker logs -f mdc_local_mdc_1

{"level":"[INFO]","ts":"Jan  25 09:58:25","logger":"VMAgent.MDC","caller":"eventhub/eventhub_sink.go:27","msg":"sending an events batch","length":1}
```



