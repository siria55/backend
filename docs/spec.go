package docs

const specJSON = `{
  "swagger": "2.0",
  "info": {
    "title": "Mars Outpost API",
    "description": "API documentation for the Mars Outpost simulation backend.",
    "version": "1.0"
  },
  "basePath": "/v1",
  "schemes": ["http"],
  "paths": {
    "/game/scene": {
      "get": {
        "tags": ["Game"],
        "summary": "获取当前火星场景配置",
        "produces": ["application/json"],
        "responses": {
          "200": {
            "description": "成功",
            "schema": {"$ref": "#/definitions/game.Scene"}
          }
        }
      }
    },
    "/system/scene": {
      "get": {
        "tags": ["System"],
        "summary": "获取 system_* 场景快照",
        "produces": ["application/json"],
        "responses": {
          "200": {
            "description": "成功",
            "schema": {"$ref": "#/definitions/game.Snapshot"}
          }
        }
      },
      "put": {
        "tags": ["System"],
        "summary": "更新 system_* 场景配置",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/server.SystemSceneUpdateRequest"}
          }
        ],
        "responses": {
          "200": {
            "description": "更新后的快照",
            "schema": {"$ref": "#/definitions/game.Snapshot"}
          },
          "400": {
            "description": "请求参数错误",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
          }
        }
      }
    },
    "/agents/{agentID}/actions": {
      "get": {
        "tags": ["Agents"],
        "summary": "列出 Agent 行为事件",
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "agentID",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "limit",
            "in": "query",
            "required": false,
            "type": "integer"
          }
        ],
        "responses": {
          "200": {
            "description": "事件列表",
            "schema": {
              "type": "array",
              "items": {"$ref": "#/definitions/action.Event"}
            }
          }
        }
      },
      "post": {
        "tags": ["Agents"],
        "summary": "记录 Agent 行为事件",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "agentID",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/server.AgentActionRequest"}
          }
        ],
        "responses": {
          "201": {
            "description": "记录成功",
            "schema": {"$ref": "#/definitions/server.StatusResponse"}
          },
          "400": {
            "description": "请求参数错误",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
          }
        }
      }
    },
    "/agents/{agentID}/state": {
      "get": {
        "tags": ["Agents"],
        "summary": "获取 Agent 行为状态",
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "agentID",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "responses": {
          "200": {
            "description": "成功",
            "schema": {"$ref": "#/definitions/action.State"}
          }
        }
      }
    }
  },
  "definitions": {
    "action.Event": {
      "type": "object",
      "properties": {
        "id": {"type": "integer"},
        "agent_id": {"type": "string"},
        "action_type": {"type": "string"},
        "payload": {"type": "object"},
        "issued_by": {"type": "string"},
        "source": {"type": "string"},
        "correlation_id": {"type": "string"},
        "result_status": {"type": "string"},
        "result_message": {"type": "string"},
        "created_at": {"type": "string", "format": "date-time"}
      }
    },
    "action.State": {
      "type": "object",
      "properties": {
        "agent_id": {"type": "string"},
        "actions": {
          "type": "array",
          "items": {"type": "string"}
        },
        "updated_at": {"type": "string", "format": "date-time"}
      }
    },
    "game.Scene": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "name": {"type": "string"},
        "grid": {"$ref": "#/definitions/game.SceneGrid"},
        "dimensions": {"$ref": "#/definitions/game.SceneDims"},
        "buildings": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.SceneBuilding"}
        },
        "agents": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.SceneAgent"}
        },
        "templates": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.BuildingTemplate"}
        }
      }
    },
    "game.Snapshot": {
      "type": "object",
      "properties": {
        "scene": {"$ref": "#/definitions/game.SceneMeta"},
        "grid": {"$ref": "#/definitions/game.SceneGrid"},
        "dimensions": {"$ref": "#/definitions/game.SceneDims"},
        "buildings": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.SceneBuilding"}
        },
        "agents": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.SceneAgent"}
        },
        "templates": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.BuildingTemplate"}
        }
      }
    },
    "game.SceneGrid": {
      "type": "object",
      "properties": {
        "cols": {"type": "integer"},
        "rows": {"type": "integer"},
        "tileSize": {"type": "integer"}
      }
    },
    "game.SceneDims": {
      "type": "object",
      "properties": {
        "width": {"type": "integer"},
        "height": {"type": "integer"}
      }
    },
    "game.SceneBuilding": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "templateId": {"type": "string"},
        "label": {"type": "string"},
        "rect": {
          "type": "array",
          "items": {"type": "integer"}
        },
        "energy": {"$ref": "#/definitions/game.SceneEnergy"}
      }
    },
    "game.SceneEnergy": {
      "type": "object",
      "properties": {
        "type": {"type": "string"},
        "capacity": {"type": "integer"},
        "current": {"type": "integer"},
        "output": {"type": "integer"},
        "rate": {"type": "integer"}
      }
    },
    "game.SceneAgent": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "label": {"type": "string"},
        "position": {
          "type": "array",
          "items": {"type": "integer"}
        },
        "color": {"type": "integer"},
        "actions": {
          "type": "array",
          "items": {"type": "string"}
        }
      }
    },
    "game.SceneMeta": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "name": {"type": "string"}
      }
    },
    "game.BuildingTemplate": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "label": {"type": "string"},
        "energy": {"$ref": "#/definitions/game.SceneEnergy"}
      }
    },
    "server.AgentActionRequest": {
      "type": "object",
      "properties": {
        "label": {"type": "string"},
        "action_type": {"type": "string"},
        "payload": {"type": "object"},
        "issued_by": {"type": "string"},
        "source": {"type": "string"},
        "correlation_id": {"type": "string"},
        "result_status": {"type": "string"},
        "result_message": {"type": "string"},
        "actions": {
          "type": "array",
          "items": {"type": "string"}
        }
      },
      "required": ["action_type"]
    },
    "server.SystemSceneUpdateRequest": {
      "type": "object",
      "properties": {
        "scene_id": {"type": "string"},
        "name": {"type": "string"},
        "grid": {"$ref": "#/definitions/server.SystemSceneUpdateGrid"},
        "dimensions": {"$ref": "#/definitions/server.SystemSceneUpdateBounds"}
      },
      "required": ["scene_id", "name", "grid", "dimensions"]
    },
    "server.SystemSceneUpdateGrid": {
      "type": "object",
      "properties": {
        "cols": {"type": "integer"},
        "rows": {"type": "integer"},
        "tileSize": {"type": "integer"}
      }
    },
    "server.SystemSceneUpdateBounds": {
      "type": "object",
      "properties": {
        "width": {"type": "integer"},
        "height": {"type": "integer"}
      }
    },
    "server.StatusResponse": {
      "type": "object",
      "properties": {
        "status": {"type": "string"}
      }
    },
    "server.ErrorResponse": {
      "type": "object",
      "properties": {
        "error": {"type": "string"}
      }
    }
  }
}`

// Spec 返回 OpenAPI/Swagger JSON 文本。
func Spec() []byte {
	return []byte(specJSON)
}
