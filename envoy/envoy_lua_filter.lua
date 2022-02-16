function rewriteUpstreamOverflowResponse(response_handle)
    local headers = response_handle:headers()
    headers:remove("x-envoy-overloaded")
    headers:replace("ms-azureml-model-error-reason", "too_many_pending_request")

    local content_length = response_handle:body():setBytes("")
    response_handle:headers():replace("content-length", content_length)
    response_handle:headers():replace("content-type", "text/plain")
end

function isUpstreamOverflow(response_handle)
    return response_handle:headers():get("x-envoy-overloaded") == "true"
end

function isUpstreamModelNotReady(response_handle)
    return response_handle:headers():get("ms-azureml-model-error-reason") == "model_not_ready"
end

function backupRequestHeaders(request_handle)
    local metadata = request_handle:metadata()
    local dynamicMetadata = request_handle:streamInfo():dynamicMetadata()
    local headers = request_handle:headers()
    local headerNames = metadata:get("RequestHeadersToPassBy")
    if headerNames then
        local headerValues = {}
        for i, v in ipairs(headerNames) do
            headerValues[v] = headers:get(v)
        end
        dynamicMetadata:set("envoy.filters.http.lua", "requestHeaders", headerValues)
    end
end

function backupAttemptCountHeader(handle)
    local headerName = "x-envoy-attempt-count"
    local headers = handle:headers()
    local attemptCount = headers:get(headerName)
    if attemptCount then
        handle:streamInfo():dynamicMetadata():set("envoy.filters.http.lua", "attemptCount", attemptCount)
        headers:remove(headerName)
    end
end

function forceBufferingResponseBody(response_handle)
    local filterName, keyName = "envoy.filters.http.lua", "isBufferingBody"
    local statusCode = response_handle:headers():get(":status")
    -- set metadata
    response_handle:streamInfo():dynamicMetadata():set(filterName, keyName, true)
    response_handle:streamInfo():dynamicMetadata():set(filterName, "originalCode", statusCode)
    -- force buffering response body
    response_handle:body()
    -- unset metadata
    response_handle:streamInfo():dynamicMetadata():set(filterName, keyName, nil)
end

function rewriteStatusNon200(response_handle)
    local headers = response_handle:headers()
    local statusCode = headers:get(":status")
    if statusCode ~= "200" then
        headers:replace("ms-azureml-model-error-reason", "model_error")
        headers:replace("ms-azureml-model-error-statuscode", statusCode)
        headers:replace(":status", "424")
    end
end

function appendEnvoyModelTime(response_handle)
    local headers = response_handle:headers()
    local modelTime = headers:get("x-envoy-upstream-service-time")
    if modelTime then
        headers:replace("ms-azureml-model-time", modelTime)
    end
end

function backupEnvoyModelTime(handle)
    local headerName = "ms-azureml-model-time"
    local headers = handle:headers()
    local modelTime = headers:get(headerName)
    if modelTime then
        handle:streamInfo():dynamicMetadata():set("envoy.filters.http.lua", "modelTime", modelTime)
        headers:remove(headerName)
    end
end

function getEnvAsBoolean(envName, defaultVal)
    local envVal = string.lower(os.getenv(envName) or "")
    local result = false
    if envVal == "true" then
        result = true
    elseif envVal == "false" then
        result = false
    elseif defaultVal ~= nil then
        result = defaultVal
    end
    return result
end

function sample(sampleRate) 
    if sampleRate >= 1.0 then
        return true
    end
    return math.random() < sampleRate
end

-- MDC Settings
local maxPayloadSize = (tonumber(os.getenv("MDC_MAX_PAYLOAD_SIZE")) or 1.0) * 1024 * 1024  -- convert to size in bytes
local maxSampleRate = (tonumber(os.getenv("MDC_SAMPLE_RATE")) or 100.0) / 100

local uuid = require "app.uuid"

function mdcSendAsync(handle, isReq)
    if isReq then
        local sampleResult = sample(maxSampleRate)
        handle:streamInfo():dynamicMetadata():set("envoy.filters.http.lua", "mdc.sample", sampleResult)
        if not sampleResult then
            do return end
        end
    else
        local sampleResult = handle:streamInfo():dynamicMetadata():get("envoy.filters.http.lua")["mdc.sample"]
        if not sampleResult then
            do return end
        end
    end

    local headers = {}
    local traceId = uuid.generate()
    
    headers["trace-id"] = traceId
    headers["ce-time"] = tostring(os.time(os.date("!*t")))
    mdcCluster = "mdc-cluster"
    headers[":authority"] = mdcCluster
    headers[":method"] = "POST"
    headers[":path"] = isReq and "/modeldata/input" or "/modeldata/output"

    for k, v in pairs(handle:headers()) do
        -- filter pseudo-headers and transfer format headers
        if not string.match(k, ":.*") then
            headers["ce-"..k] = v
        elseif k == ":method" then
            headers["ce-method"] = v
        elseif k == ":path" then
            headers["ce-path"] = v
        end
    end

    local seq = 0

    for chunk in handle:bodyChunks() do
        local len = chunk:length()
        local body = chunk:getBytes(0, len)
        headers["seq"] = tostring(seq)

        handle:logDebug("sending chunk to mdc #"..tostring(seq).." size:".. tostring(chunk:length()))
        handle:httpCall(mdcCluster, headers, body, 5000, true)
        seq = seq + 1
    end    

    if seq == 0  then
        -- No body for this payload
        handle:httpCall(mdcCluster, headers, nil, 5000, true)
    end
end

