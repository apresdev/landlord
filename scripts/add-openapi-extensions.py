#!/usr/bin/env python3

import sys

import json

input_filename = sys.argv[1]

with open(input_filename) as file:
    openapi = json.load(file)

for path in openapi["paths"]:
    for method in openapi["paths"][path]:
        openapi["paths"][path][method]["x-amazon-apigateway-integration"] = {
            "connectionId": "${vpc_link_connection_id}",
            "connectionType": "VPC_LINK",
            "httpMethod": "any",
            "type": "http_proxy",
            "uri": "${nlb_uri}" + path,
            "payloadFormatVersion": "1.0"
        }
    # see https://docs.aws.amazon.com/apigateway/latest/developerguide/enable-cors-for-resource-using-swagger-importer-tool.html
    openapi["components"]["responses"]["CorsResponse"] = {
      "headers": {
        "Access-Control-Allow-Origin": {
          "schema": {
            "type": "string"
          }
        },
        "Access-Control-Allow-Methods": {
          "schema": {
            "type": "string"
          }
        },
        "Access-Control-Allow-Headers": {
          "schema": {
            "type": "string"
          }
        }
      },
      "content": {}
    }
    first = next(iter(openapi["paths"][path].values()))
    if "parameters" in first.keys():
      parameters = first["parameters"]
    else:
      parameters = []
    openapi["paths"][path]["options"] = {
      "parameters": parameters,
      "responses": {
        "200": {
          "$ref": "#/components/responses/CorsResponse"
        }
      },
      "x-amazon-apigateway-integration": {
        "type": "mock",
        "requestTemplates": {
          "application/json": "{\"statusCode\": 200}"
        },
        "passthroughBehavior": "never",
        "responses": {
          "default": {
            "statusCode": "200",
            "responseParameters": {
              # todo: how do we inject the right allowed origin?
              "method.response.header.Access-Control-Allow-Headers": "'*'",
              "method.response.header.Access-Control-Allow-Methods": "'*'",
              "method.response.header.Access-Control-Allow-Origin": "'*'"
            }
          }
        }
      }
    }

print(json.dumps(openapi, indent=2))
