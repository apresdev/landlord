#!/usr/bin/env python3

import json
import sys


def resolve(node, openapi_spec):
    """
    Resolves an optional OpenAPI $ref from an OpenAPI node.

    If the object contains a "$ref" then the corresponding node is lookup up in the openAPI specification, otherwise, the node itself is returned.
    :param node: The OpenAPI node to resolve.
    :param openapi_spec: The complete OpenAPI specification.
    :return:  The resolved OpenAPI node.
    """

    if "$ref" in node.keys():
        parts = node["$ref"].split("/")
        result = openapi_spec
        for part in parts:
            if "#" != part:
                result = result[part]

        return result
    else:
        return node


def map_openapi_parameters(openapi_parameters, openapi_spec):
    """
    Converts the OpenAPI parameters object from an OpenAPI operation into the corresponding `x-amazon-apigateway-integration` requestParameters.

    See: https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-swagger-extensions-integration-requestParameters.html.
    :param openapi_parameters:  The OpenAPI parameters collection.
    :param openapi_spec: The complete OpenAPI specification.
    :return: the `x-amazon-apigateway-integration` requestParameters object.
    """
    result = {}
    for param in openapi_parameters:
        request_parameter = resolve(param, openapi_spec)
        match request_parameter["in"]:
            case "path":
                result["integration.request.path." + request_parameter["name"]] = "method.request.path." + request_parameter["name"]
    # doesn't look like we need this for query/header parameters, leads to duplicated values
    #       case "query":
    #         result["integration.request.querystring." + request_parameter["name"]] = "method.request.querystring." + request_parameter["name"]
    #       case "header":
    #         result["integration.request.header." + request_parameter["name"]] = "method.request.header." + request_parameter["name"]

    return result


filename = sys.argv[1]
with open(filename) as file:
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

        if "parameters" in openapi["paths"][path][method]:
            openapi["paths"][path][method]["x-amazon-apigateway-integration"]["requestParameters"] = map_openapi_parameters(
                openapi["paths"][path][method]["parameters"], openapi)

    # see https://docs.aws.amazon.com/apigateway/latest/developerguide/enable-cors-for-resource-using-swagger-importer-tool.html
    openapi["components"]["responses"]["CorsResponse"] = {
        "description": "Default response for CORS method",
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
