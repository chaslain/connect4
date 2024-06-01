Connect4 teleegram bot, hastily thrown together in golang.
The bot uses sqlite to store state.
It reads the following configurations from a yaml file that must be created:
resources/config.yaml

|Config | Type | Description |
|-------|------|-------------|
| telegram_bot_token | string | The bot token provieded by telegram. |
| schema_dsn | string | the sqlite dsn. View the [Sqlite repo](https://github.com/mattn/go-sqlite3) for more info.|
| debug | boolean | Print debug logs- mostly deprecated. In most situations this is ignored. |
| base_elo | integer | The rating number new players begin at. |
| elo_k | integer | the "k" value in the elo formula. The higher the number, the more volatile the elo. |
| public_key_path | string | The public key to send to the webhook so telegram knows what key to expect. |
| webhook_url | string | the url the TLS proxy server is listening on, with the port. |
| kill_age | integer | the amount of minutes without a move before you can claim a win. |