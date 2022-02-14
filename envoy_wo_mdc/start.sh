#!/bin/bash

baseId=$1
flag=""
concurrency=""
if [ "${MIR_MESH_DISABLE,,}" = "true" ]; then
  echo "MIR_MESH_DISABLE on"
  flag="-nomesh"
elif [ "${ENABLE_TRITON_MULTI_MODEL,,}" = "true" ]; then
  echo "ENABLE_TRITON_MULTI_MODEL on"
  flag="-manymodel"
fi

if [ "$baseId" = "2" ]; then
  flag="${flag}-layer2"
  concurrency="--concurrency 1"
fi

echo "flag=$flag"
echo "set MIR_MTLS_DISABLE=$MIR_MTLS_DISABLE"

envoyConfig="/app/envoy${flag}.yaml"

# replace env
tmpConfig="/app/envoy_final_${baseId}.yaml"
cp $envoyConfig $tmpConfig

if [ -z "$MODEL_REQUEST_TIMEOUT" ]; then
  MODEL_REQUEST_TIMEOUT="600"
fi
echo "set MODEL_REQUEST_TIMEOUT=$MODEL_REQUEST_TIMEOUT"
sed -i "s/{{MODEL_REQUEST_TIMEOUT}}/$MODEL_REQUEST_TIMEOUT/g" $tmpConfig

echo "set MIR_ENVOY_RETRY_ATTEMPTS=$MIR_ENVOY_RETRY_ATTEMPTS"
sed -i "s/{{MIR_ENVOY_RETRY_ATTEMPTS}}/$MIR_ENVOY_RETRY_ATTEMPTS/g" $tmpConfig

# replace load balance algo
echo "set ENVOY_MESH_LOAD_BALANCE_ALGO=$ENVOY_MESH_LOAD_BALANCE_ALGO"
sed -i "s/{{ENVOY_MESH_LOAD_BALANCE_ALGO}}/$ENVOY_MESH_LOAD_BALANCE_ALGO/g" $tmpConfig

# replace idle timeout and max connection duration
echo "set MIR_SCORING_PROXY_IDLE_TIMEOUT=$MIR_SCORING_PROXY_IDLE_TIMEOUT, MIR_SCORING_PROXY_MAX_CONNECTION_DURATION=$MIR_SCORING_PROXY_MAX_CONNECTION_DURATION"
sed -i "s/{{MIR_SCORING_PROXY_IDLE_TIMEOUT}}/$MIR_SCORING_PROXY_IDLE_TIMEOUT/g" $tmpConfig
sed -i "s/{{MIR_SCORING_PROXY_MAX_CONNECTION_DURATION}}/$MIR_SCORING_PROXY_MAX_CONNECTION_DURATION/g" $tmpConfig

LOG_SAMPLE_RATIO=100
if [ "${MIR_ENVOY_HIGH_QPS_OPTIMIZED,,}" = "true" ]; then
  echo "MIR_ENVOY_HIGH_QPS_OPTIMIZED on"
  # disable second layer logs and worker count limit
  LOG_SAMPLE_RATIO=0
  concurrency=""
fi
sed -i "s/{{LOG_SAMPLE_RATIO}}/$LOG_SAMPLE_RATIO/g" $tmpConfig

if [ -z "$ENABLE_MDC" ]; then
  ENABLE_MDC="false"
fi
if [ "${ENABLE_MDC,,}" = "true" ]; then
  sed -i -E "s/--\\s*mdcSendAsync/mdcSendAsync/g" $tmpConfig
fi
echo "set ENABLE_MDC=$ENABLE_MDC"

level="${MIR_ENVOY_LOG_LEVEL:-info}"
echo "set Envoy log level=$level"

/usr/local/bin/envoy -c $tmpConfig --base-id 3 $concurrency --log-level $level --drain-time-s 0 --drain-strategy immediate --bootstrap-version 3
