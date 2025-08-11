package migrations

import (
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// update collections
		jsonData := `[
	{
		"id": "elngm8x1l60zi2v",
		"listRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"viewRule": "",
		"createRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"updateRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"deleteRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"name": "alerts",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "hn5ly3vi",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "user",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"cascadeDelete": true,
				"collectionId": "2hz5ncl8tizk5nx",
				"hidden": false,
				"id": "g5sl3jdg",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "system",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "zj3ingrv",
				"maxSelect": 1,
				"name": "name",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "select",
				"values": [
					"Status",
					"Ping"
				]
			},
			{
				"hidden": false,
				"id": "o2ablxvn",
				"max": null,
				"min": null,
				"name": "value",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "fstdehcq",
				"max": 60,
				"min": null,
				"name": "min",
				"onlyInt": true,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "6hgdf6hs",
				"name": "triggered",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "bool"
			},
			{
				"hidden": false,
				"id": "autodate2990389176",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`" + `idx_MnhEt21L5r` + "`" + ` ON ` + "`" + `alerts` + "`" + ` (\n  ` + "`" + `user` + "`" + `,\n  ` + "`" + `system` + "`" + `,\n  ` + "`" + `name` + "`" + `\n)"
		],
		"system": false
	},
	{
		"id": "pbc_1697146157",
		"listRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"createRule": null,
		"updateRule": null,
		"deleteRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"name": "alerts_history",
		"type": "base",
		"fields": [
			{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text3208210256",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": true,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "relation2375276105",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "user",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": true,
					"collectionId": "2hz5ncl8tizk5nx",
					"hidden": false,
					"id": "relation3377271179",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "system",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text2466471794",
					"max": 0,
					"min": 0,
					"name": "alert_id",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1579384326",
					"max": 0,
					"min": 0,
					"name": "name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "number494360628",
					"max": null,
					"min": null,
					"name": "value",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "date2276568630",
					"max": "",
					"min": "",
					"name": "resolved",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				}
		],
		"indexes": [
			"CREATE INDEX ` + "`" + `idx_YdGnup5aqB` + "`" + ` ON ` + "`" + `alerts_history` + "`" + ` (` + "`" + `user` + "`" + `)",
			"CREATE INDEX ` + "`" + `idx_taLet9VdME` + "`" + ` ON ` + "`" + `alerts_history` + "`" + ` (` + "`" + `created` + "`" + `)"
		],
		"system": false
	},
	{
		"id": "pbc_3663931638",
		"listRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"createRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id && @request.auth.role != \"readonly\"",
		"updateRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id && @request.auth.role != \"readonly\"",
		"deleteRule": null,
		"name": "fingerprints",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{9}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 9,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "2hz5ncl8tizk5nx",
				"hidden": false,
				"id": "relation3377271179",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "system",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"autogeneratePattern": "[a-zA-Z9-9]{20}",
				"hidden": false,
				"id": "text1597481275",
				"max": 255,
				"min": 9,
				"name": "token",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "text4228609354",
				"max": 255,
				"min": 9,
				"name": "fingerprint",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [
			"CREATE INDEX ` + "`" + `idx_p9qZlu26po` + "`" + ` ON ` + "`" + `fingerprints` + "`" + ` (` + "`" + `token` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_ngboulGMYw` + "`" + ` ON ` + "`" + `fingerprints` + "`" + ` (` + "`" + `system` + "`" + `)"
		],
		"system": false
	},
	{
		"id": "4afacsdnlu8q8r2",
		"listRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"viewRule": null,
		"createRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"updateRule": "@request.auth.id != \"\" && user.id = @request.auth.id",
		"deleteRule": null,
		"name": "user_settings",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "d5vztyxa",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "user",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "xcx4qgqq",
				"maxSize": 2000000,
				"name": "settings",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "autodate2990389176",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`" + `idx_30Lwgf2` + "`" + ` ON ` + "`" + `user_settings` + "`" + ` (` + "`" + `user` + "`" + `)"
		],
		"system": false
	},
	{
		"id": "2hz5ncl8tizk5nx",
		"listRule": "@request.auth.id != \"\" && users.id ?= @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && users.id ?= @request.auth.id",
		"createRule": "@request.auth.id != \"\" && users.id ?= @request.auth.id && @request.auth.role != \"readonly\"",
		"updateRule": "@request.auth.id != \"\" && users.id ?= @request.auth.id && @request.auth.role != \"readonly\"",
		"deleteRule": "@request.auth.id != \"\" && users.id ?= @request.auth.id && @request.auth.role != \"readonly\"",
		"name": "systems",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "7xloxkwk",
				"max": 0,
				"min": 0,
				"name": "name",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "waj7seaf",
				"maxSelect": 1,
				"name": "status",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "select",
				"values": [
					"up",
					"down",
					"paused",
					"pending"
				]
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "ve781smf",
				"max": 0,
				"min": 0,
				"name": "host",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "pij0k2jk",
				"max": 0,
				"min": 0,
				"name": "port",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "qoq64ntl",
				"maxSize": 2000000,
				"name": "info",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"cascadeDelete": true,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "jcarjnjj",
				"maxSelect": 2147483647,
				"minSelect": 0,
				"name": "users",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},

			{
				"hidden": false,
				"id": "autodate2990389176",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [],
		"system": false
	},
	{
		"id": "system_stats_collection_id",
		"listRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"name": "ping_stats",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "2hz5ncl8tizk5nx",
				"hidden": false,
				"id": "system_relation_id",
				"maxSelect": 1,
				"minSelect": 1,
				"name": "system",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "host_text_id",
				"maxLength": 255,
				"name": "host",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "packet_loss_number_id",
				"name": "packet_loss",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "min_rtt_number_id",
				"name": "min_rtt",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "max_rtt_number_id",
				"name": "max_rtt",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "avg_rtt_number_id",
				"name": "avg_rtt",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "created_date_id",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "updated_date_id",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [],
		"system": false
	},
	{
		"id": "dns_stats_collection_id",
		"listRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"name": "dns_stats",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "2hz5ncl8tizk5nx",
				"hidden": false,
				"id": "system_relation_id",
				"maxSelect": 1,
				"minSelect": 1,
				"name": "system",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "domain_text_id",
				"maxLength": 255,
				"name": "domain",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "server_text_id",
				"maxLength": 255,
				"name": "server",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "type_text_id",
				"maxLength": 10,
				"name": "type",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "status_text_id",
				"maxLength": 20,
				"name": "status",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "lookup_time_number_id",
				"name": "lookup_time",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "error_code_text_id",
				"maxLength": 100,
				"name": "error_code",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "created_date_id",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "updated_date_id",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [],
		"system": false
	},
	{
		"id": "http_stats_collection_id",
		"listRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"name": "http_stats",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "2hz5ncl8tizk5nx",
				"hidden": false,
				"id": "system_relation_id",
				"maxSelect": 1,
				"minSelect": 1,
				"name": "system",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "url_text_id",
				"maxLength": 500,
				"name": "url",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "status_text_id",
				"maxLength": 20,
				"name": "status",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "response_time_number_id",
				"name": "response_time",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "status_code_number_id",
				"name": "status_code",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "error_code_text_id",
				"maxLength": 100,
				"name": "error_code",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "created_date_id",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "updated_date_id",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [],
		"system": false
	},
	{
		"id": "speedtest_stats_collection_id",
		"listRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"name": "speedtest_stats",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "2hz5ncl8tizk5nx",
				"hidden": false,
				"id": "system_relation_id",
				"maxSelect": 1,
				"minSelect": 1,
				"name": "system",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "server_id_text_id",
				"maxLength": 100,
				"name": "server_id",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "status_text_id",
				"maxLength": 20,
				"name": "status",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "download_speed_number_id",
				"name": "download_speed",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "upload_speed_number_id",
				"name": "upload_speed",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "latency_number_id",
				"name": "latency",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "error_code_text_id",
				"maxLength": 100,
				"name": "error_code",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "ping_jitter_number_id",
				"name": "ping_jitter",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "ping_low_number_id",
				"name": "ping_low",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "ping_high_number_id",
				"name": "ping_high",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "download_bytes_number_id",
				"name": "download_bytes",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "download_elapsed_number_id",
				"name": "download_elapsed",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "download_latency_iqm_number_id",
				"name": "download_latency_iqm",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "download_latency_low_number_id",
				"name": "download_latency_low",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "download_latency_high_number_id",
				"name": "download_latency_high",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "download_latency_jitter_number_id",
				"name": "download_latency_jitter",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "upload_bytes_number_id",
				"name": "upload_bytes",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "upload_elapsed_number_id",
				"name": "upload_elapsed",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "upload_latency_iqm_number_id",
				"name": "upload_latency_iqm",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "upload_latency_low_number_id",
				"name": "upload_latency_low",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "upload_latency_high_number_id",
				"name": "upload_latency_high",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "upload_latency_jitter_number_id",
				"name": "upload_latency_jitter",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "packet_loss_number_id",
				"name": "packet_loss",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "isp_text_id",
				"maxLength": 200,
				"name": "isp",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "interface_external_ip_text_id",
				"maxLength": 50,
				"name": "interface_external_ip",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "server_name_text_id",
				"maxLength": 200,
				"name": "server_name",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "server_location_text_id",
				"maxLength": 200,
				"name": "server_location",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "server_country_text_id",
				"maxLength": 100,
				"name": "server_country",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "server_host_text_id",
				"maxLength": 200,
				"name": "server_host",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},

			{
				"hidden": false,
				"id": "server_ip_text_id",
				"maxLength": 50,
				"name": "server_ip",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},

			{
				"hidden": false,
				"id": "created_date_id",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "updated_date_id",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [],
		"system": false
	},
	{
		"id": "_pb_users_auth_",
		"listRule": "id = @request.auth.id",
		"viewRule": "id = @request.auth.id",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"name": "users",
		"type": "auth",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cost": 10,
				"hidden": true,
				"id": "password901924565",
				"max": 0,
				"min": 8,
				"name": "password",
				"pattern": "",
				"presentable": false,
				"required": true,
				"system": true,
				"type": "password"
			},
			{
				"autogeneratePattern": "[a-zA-Z0-9_]{50}",
				"hidden": true,
				"id": "text2504183744",
				"max": 60,
				"min": 30,
				"name": "tokenKey",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"exceptDomains": null,
				"hidden": false,
				"id": "email3885137012",
				"name": "email",
				"onlyDomains": null,
				"presentable": false,
				"required": true,
				"system": true,
				"type": "email"
			},
			{
				"hidden": false,
				"id": "bool1547992806",
				"name": "emailVisibility",
				"presentable": false,
				"required": false,
				"system": true,
				"type": "bool"
			},
			{
				"hidden": false,
				"id": "bool256245529",
				"name": "verified",
				"presentable": false,
				"required": false,
				"system": true,
				"type": "bool"
			},
			{
				"autogeneratePattern": "users[0-9]{6}",
				"hidden": false,
				"id": "text4166911607",
				"max": 150,
				"min": 3,
				"name": "username",
				"pattern": "^[\\w][\\w\\.\\-]*$",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "qkbp58ae",
				"maxSelect": 1,
				"name": "role",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "select",
				"values": [
					"user",
					"admin",
					"readonly"
				]
			},
			{
				"hidden": false,
				"id": "autodate2990389176",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`" + `__pb_users_auth__username_idx` + "`" + ` ON ` + "`" + `users` + "`" + ` (username COLLATE NOCASE)",
			"CREATE UNIQUE INDEX ` + "`" + `__pb_users_auth__email_idx` + "`" + ` ON ` + "`" + `users` + "`" + ` (` + "`" + `email` + "`" + `) WHERE ` + "`" + `email` + "`" + ` != ''",
			"CREATE UNIQUE INDEX ` + "`" + `__pb_users_auth__tokenKey_idx` + "`" + ` ON ` + "`" + `users` + "`" + ` (` + "`" + `tokenKey` + "`" + `)"
		],
		"system": false,
		"authRule": "verified=true",
		"manageRule": null
	},
	{
		"id": "monitoring_config_collection",
		"listRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && system.users.id ?= @request.auth.id",
		"createRule": "@request.auth.id != \"\" && @request.auth.role != \"readonly\"",
		"updateRule": "@request.auth.id != \"\" && @request.auth.role != \"readonly\"",
		"deleteRule": "@request.auth.id != \"\" && @request.auth.role != \"readonly\"",
		"name": "monitoring_config",
		"type": "base",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "2hz5ncl8tizk5nx",
				"hidden": false,
				"id": "system_relation",
				"maxSelect": 1,
				"minSelect": 1,
				"name": "system",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation",
				"unique": true
			},
			{
				"hidden": false,
				"id": "ping",
				"maxSize": 2000000,
				"name": "ping",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "dns",
				"maxSize": 2000000,
				"name": "dns",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "http",
				"maxSize": 2000000,
				"name": "http",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "speedtest",
				"maxSize": 2000000,
				"name": "speedtest",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "autodate2990389176",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"indexes": [],
		"system": false
	}
]`

		err := app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
		if err != nil {
			return err
		}

		// Get all systems that don't have fingerprint records
		var systemIds []string
		err = app.DB().NewQuery(`
			SELECT s.id FROM systems s
			LEFT JOIN fingerprints f ON s.id = f.system
			WHERE f.system IS NULL
		`).Column(&systemIds)
		if err != nil {
			return err
		}
		// Create fingerprint records with unique UUID tokens for each system
		for _, systemId := range systemIds {
			token := uuid.New().String()
			_, err = app.DB().NewQuery(`
				INSERT INTO fingerprints (system, token)
				VALUES ({:system}, {:token})
			`).Bind(map[string]any{
				"system": systemId,
				"token":  token,
			}).Execute()
			if err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}
