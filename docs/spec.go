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
    "/game/scene/stream": {
      "get": {
        "tags": ["Game"],
        "summary": "订阅火星场景 WebSocket 流",
        "produces": ["application/json"],
        "responses": {
          "101": {
            "description": "WebSocket Upgrade"
          }
        }
      }
    },
    "/game/scene/buildings/{buildingID}/energy": {
      "post": {
        "tags": ["Game"],
        "summary": "更新指定建筑的当前能量值",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "buildingID",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/server.BuildingEnergyUpdateRequest"}
          }
        ],
        "responses": {
          "200": {
            "description": "更新后的建筑数据",
            "schema": {"$ref": "#/definitions/game.SceneBuilding"}
          },
          "400": {
            "description": "请求参数错误",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
          }
        }
      }
    },
    "/game/scene/agents/{agentID}/position": {
      "put": {
        "tags": ["Game"],
        "summary": "更新 Agent 运行时坐标",
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
            "schema": {"$ref": "#/definitions/server.AgentPositionUpdateRequest"}
          }
        ],
        "responses": {
          "200": {
            "description": "更新后的 Agent 数据",
            "schema": {"$ref": "#/definitions/game.SceneAgent"}
          },
          "400": {
            "description": "请求参数错误",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
          }
        }
      }
    },
    "/game/scene/agents/{agentID}/behaviors/maintain-energy": {
      "post": {
        "tags": ["Game"],
        "summary": "保持电量不减少（自动建造太阳能塔）",
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
            "description": "操作结果",
            "schema": {"$ref": "#/definitions/server.MaintainEnergyResponse"}
          },
          "400": {
            "description": "请求参数错误",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
          },
          "409": {
            "description": "缺少可用空间或资源冲突",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
          },
          "424": {
            "description": "缺少太阳能塔模板",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
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
    "/system/templates/buildings/{id}": {
      "put": {
        "tags": ["System"],
        "summary": "更新系统建筑模板",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/server.TemplateBuildingRequest"}
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
    "/system/templates/agents/{id}": {
      "put": {
        "tags": ["System"],
        "summary": "更新系统 Agent 模板",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/server.TemplateAgentRequest"}
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
    "/system/scene/buildings/{id}": {
      "put": {
        "tags": ["System"],
        "summary": "更新场景建筑实例",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/server.SceneBuildingRequest"}
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
      },
      "delete": {
        "tags": ["System"],
        "summary": "删除场景建筑实例",
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "string"
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
    "/system/scene/buildings/preview": {
      "get": {
        "tags": ["System"],
        "summary": "预览场景建筑列表",
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "sceneId",
            "in": "query",
            "required": false,
            "type": "string",
            "description": "可选的场景 ID，缺省为当前运行中的场景"
          },
          {
            "name": "limit",
            "in": "query",
            "required": false,
            "type": "integer",
            "format": "int32",
            "description": "限制返回的建筑数量"
          }
        ],
        "responses": {
          "200": {
            "description": "预览数据",
            "schema": {
              "type": "object",
              "properties": {
                "sceneId": {"type": "string"},
                "count": {"type": "integer", "format": "int32"},
                "buildings": {
                  "type": "array",
                  "items": {"$ref": "#/definitions/game.SceneBuilding"}
                }
              }
            }
          },
          "400": {
            "description": "请求参数错误",
            "schema": {"$ref": "#/definitions/server.ErrorResponse"}
          }
        }
      }
    },
    "/system/scene/agents/{id}": {
      "put": {
        "tags": ["System"],
        "summary": "更新场景 Agent 实例",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/server.SceneAgentRequest"}
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
        "buildingTemplates": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.BuildingTemplate"}
        },
        "agentTemplates": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.AgentTemplate"}
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
        "buildingTemplates": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.BuildingTemplate"}
        },
        "agentTemplates": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.AgentTemplate"}
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
    "game.AgentTemplate": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "label": {"type": "string"},
        "color": {"type": "integer"},
        "position": {
          "type": "array",
          "items": {"type": "integer"}
        }
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
    "server.TemplateEnergyRequest": {
      "type": "object",
      "properties": {
        "type": {"type": "string"},
        "capacity": {"type": "integer"},
        "current": {"type": "integer"},
        "output": {"type": "integer"},
        "rate": {"type": "integer"}
      }
    },
    "server.TemplateBuildingRequest": {
      "type": "object",
      "properties": {
        "label": {"type": "string"},
        "energy": {"$ref": "#/definitions/server.TemplateEnergyRequest"}
      },
      "required": ["label"]
    },
    "server.TemplateAgentRequest": {
      "type": "object",
      "properties": {
        "label": {"type": "string"},
        "color": {"type": "integer"},
        "defaultPosition": {
          "type": "array",
          "items": {"type": "integer"},
          "maxItems": 2,
          "minItems": 2
        }
      },
      "required": ["label"]
    },
    "server.SceneBuildingRequest": {
      "type": "object",
      "properties": {
        "label": {"type": "string"},
        "templateId": {"type": "string"},
        "rect": {
          "type": "array",
          "items": {"type": "integer"},
          "maxItems": 4,
          "minItems": 4
        },
        "energy": {"$ref": "#/definitions/server.TemplateEnergyRequest"}
      },
      "required": ["label", "rect"]
    },
    "server.SceneAgentRequest": {
      "type": "object",
      "properties": {
        "label": {"type": "string"},
        "templateId": {"type": "string"},
        "position": {
          "type": "array",
          "items": {"type": "integer"},
          "maxItems": 2,
          "minItems": 2
        },
        "color": {"type": "integer"},
        "actions": {
          "type": "array",
          "items": {"type": "string"}
        }
      },
      "required": ["label", "position"]
    },
    "server.MaintainEnergyResponse": {
      "type": "object",
      "properties": {
        "scene": {"$ref": "#/definitions/game.Scene"},
        "created": {
          "type": "array",
          "items": {"$ref": "#/definitions/game.SceneBuilding"}
        },
        "netFlowBefore": {"type": "number"},
        "netFlowAfter": {"type": "number"},
        "towersBuilt": {"type": "integer"},
        "relocation": {"$ref": "#/definitions/game.AgentRelocation"}
      }
    },
    "game.AgentRelocation": {
      "type": "object",
      "properties": {
        "id": {"type": "string"},
        "position": {
          "type": "array",
          "items": {"type": "number"},
          "minItems": 2,
          "maxItems": 2
        }
      },
      "required": ["id", "position"]
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
    },
    "server.BuildingEnergyUpdateRequest": {
      "type": "object",
      "properties": {
        "current": {"type": "number"}
      },
      "required": ["current"]
    },
    "server.AgentPositionUpdateRequest": {
      "type": "object",
      "properties": {
        "x": {"type": "number"},
        "y": {"type": "number"}
      },
      "required": ["x", "y"]
    }
  }
}`

// Spec 返回 OpenAPI/Swagger JSON 文本。
func Spec() []byte {
	return []byte(specJSON)
}
