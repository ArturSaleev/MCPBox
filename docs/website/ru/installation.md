# Установка и настройка MCPBox

[Оглавление](README.md) | [English](../en/installation.md)

В этом руководстве описано стандартное on-premises-развёртывание MCPBox.

## Требования

- поддерживаемый сервер macOS, Linux или Windows;
- права для запуска приложения и открытия выбранных TCP-портов;
- для опционального SSO: OIDC-приложение, зарегистрированное у провайдера идентификации;
- для опциональной базы знаний: доступный экземпляр RagBox;
- для опциональной runtime-проверки доступа: доступный экземпляр PolicyBox.

## 1. Скачайте и проверьте бинарник

Скачайте бинарник своей платформы из [официального релиза](https://github.com/ArturSaleev/MCPBox/releases/tag/v1.3.0) в отдельный доступный для записи каталог. После проверки файла с исходным именем переименуйте его в `MCPBox` (macOS/Linux) или `MCPBox.exe` (Windows).


Имя исполняемого файла:

- macOS / Linux: `MCPBox`;
- Windows: `MCPBox.exe`.


Перед запуском выполните [проверку SHA-256 и ознакомьтесь с предупреждениями ОС](release-verification.md). Go и Node.js для готового бинарника не нужны. Выберите amd64 для Intel/AMD или arm64 для ARM; установите отдельные runtime только для тех MCP/CLI-интеграций, которым они нужны.

## 2. Лицензия открытого ПО

MCPBox uses Apache-2.0. No activation is required.

## 3. Создайте первого администратора

Перед первым запуском выполните команду из директории приложения:

```bash
./MCPBox admin create \
  --email admin@example.com \
  --password 'change-this-password' \
  --name 'System Administrator'
```

Пароль должен состоять минимум из восьми символов. Сохраните его в защищённом менеджере паролей и замените пример уникальным паролем.

В Windows PowerShell:

```powershell
.\MCPBox.exe admin create --email admin@example.com --password 'change-this-password' --name 'System Administrator'
```

## 4. Запустите MCPBox

Запустите файл из директории установки:

```bash
./MCPBox
```

Откройте интерфейс администрирования: [http://127.0.0.1:38181/](http://127.0.0.1:38181/). Войдите с учётной записью, созданной на предыдущем шаге.

Если сервер администрируется удалённо, используйте SSH-туннель, reverse proxy либо осознанно измените host для admin listener. Не открывайте интерфейс администрирования напрямую в публичный интернет.

## Настройка listeners

MCPBox использует следующие значения по умолчанию:

| Настройка | Значение | Назначение |
| --- | --- | --- |
| Admin host | `127.0.0.1` | веб-интерфейс администрирования и Admin API |
| Admin port | `38181` | веб-интерфейс администрирования и Admin API |
| MCP host | `127.0.0.1` | listener MCP проектов |
| MCP port | `38182` | listener MCP проектов |

Переопределить значения можно переменными окружения:

```bash
MCPBOX_HOST=127.0.0.1 \
MCPBOX_PORT=39181 \
MCPBOX_MCP_HOST=0.0.0.0 \
MCPBOX_MCP_PORT=39182 \
./MCPBox
```

Или параметрами командной строки:

```bash
./MCPBox \
  --host 127.0.0.1 --port 39181 \
  --mcp-host 0.0.0.0 --mcp-port 39182
```

Адрес MCP проекта имеет вид:

```text
http://<server>:<mcp-port>/mcp/<project-token>
```

Project token берётся из интерфейса MCPBox. В production публикуйте endpoint через HTTPS и ограничивайте сетевой доступ в соответствии с политикой безопасности.

## Bootstrap token

Для аварийного/bootstrap-доступа к интерфейсу администрирования можно задать `MCPBOX_PRO_BOOTSTRAP_TOKEN`:

```bash
MCPBOX_PRO_BOOTSTRAP_TOKEN='replace-with-a-long-random-secret' ./MCPBox
```

Используйте длинный уникальный секрет и защищайте его как учётные данные. Для повседневного администрирования предпочтительнее локальные пользователи или SSO.

## Опционально: OIDC single sign-on

Generic OIDC SSO включается, когда заданы следующие переменные:

```bash
MCPBOX_PRO_OIDC_ISSUER_URL=https://id.example.com \
MCPBOX_PRO_OIDC_CLIENT_ID=your-client-id \
MCPBOX_PRO_OIDC_CLIENT_SECRET=your-client-secret \
MCPBOX_PRO_OIDC_REDIRECT_URL=https://mcpbox.example.com/api/pro/sso/callback \
./MCPBox
```

Дополнительные настройки:

- `MCPBOX_PRO_OIDC_PROVIDER_NAME`;
- `MCPBOX_PRO_OIDC_SCOPES`;
- `MCPBOX_PRO_OIDC_ALLOWED_HD`;
- `MCPBOX_PRO_OIDC_DEFAULT_ROLE`;
- `MCPBOX_PRO_OIDC_SESSION_DAYS`;
- `MCPBOX_PRO_OIDC_AUTO_CREATE_USERS`.

Точно зарегистрируйте redirect URL у провайдера идентификации. Для локальной установки обычно используется `http://127.0.0.1:38181/api/pro/sso/callback`.

## Опциональные интеграции

После входа настройте интеграции через интерфейс администрирования.

- **RagBox** предоставляет ingestion базы знаний, vector connections и retrieval. MCPBox проверяет настроенное соединение с RagBox, прежде чем считать интеграцию активной.
- **PolicyBox** предоставляет runtime-проверку идентификации и доступа. Если для проекта включена проверка идентификации, доступ к его MCP endpoint требует действующий runtime bearer token, соответствующий project binding.

Храните сервисные credentials раздельно и выдавайте каждой интеграции только необходимые права.

## Обновление MCPBox

1. Перед обновлением сделайте резервную копию директории установки и базы данных.
2. Остановите работающий процесс MCPBox.
3. Замените исполняемый файл на новую версию.
4. Запустите MCPBox и проверьте админ-интерфейс и нужные project endpoints.

Если новая версия меняет схему развёртывания, обновите правила firewall, конфигурацию reverse proxy и OIDC redirect URLs.

## Устранение неполадок

| Симптом | Что проверить |
| --- | --- |
| UI недоступен | Убедитесь, что процесс запущен и корректны `MCPBOX_HOST` / `MCPBOX_PORT`. |
| Не удаётся войти | Проверьте учётные данные локального администратора или OIDC-переменные и redirect URL. |
| MCP-клиенты не подключаются | Проверьте MCP host/port, firewall, reverse proxy, project token и требование runtime token от PolicyBox. |

Настройки по умолчанию выше относятся к текущим исходникам. Опубликованный v1.3.0 использует прежний MCP bind `0.0.0.0`; до обновления явно задавайте `--mcp-host 127.0.0.1`. Удалённый доступ включается через `--mcp-host 0.0.0.0` или `MCPBOX_MCP_HOST=0.0.0.0`. [Сеть и изоляция проектов](network-access.md).
