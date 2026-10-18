# Подключение проекта MCPBox к ChatGPT

Это руководство позволяет ChatGPT использовать tools, настроенные внутри одного проекта MCPBox.

## Что нужно заранее

- работающий MCPBox project хотя бы с одним проверенным MCP server;
- публичный HTTPS-адрес MCP endpoint, например `https://mcp.example.com/mcp/<project-token>`;
- включённая project **Bearer authentication**;
- доступ к developer mode или custom apps в вашем тарифе/workspace ChatGPT.

ChatGPT подключается к удалённым MCP servers, а не напрямую к приватному localhost или on-premises-адресу. Если сервер MCPBox приватный, используйте одобренный secure tunnel или сетевую схему, а не открывайте admin UI в интернет. Актуальные требования OpenAI описаны в [справке ChatGPT](https://help.openai.com/en/articles/12584461-developer-mode-and-full-mcp-connectors-in-chatgpt-beta).

## Шаг 1. Подготовьте проект MCPBox

1. Откройте проект в MCPBox.
2. Включите **Bearer authentication**.
3. Убедитесь, что project MCP endpoint доступен по публичному HTTPS.
4. Не раскрывайте project token и OAuth client token: не вставляйте их в prompt или screenshot.

В MCPBox есть OAuth discovery и authorization endpoints для этого flow. Для каждого подключения создаётся отдельный OAuth-пользователь со своим callback URL и token. Это позволяет подключить к одному проекту несколько пользователей или ChatGPT apps и видеть их имена в **Logs**.
Protected-resource metadata публикуется только для проекта с включённой **Bearer authentication**. Публичный MCP-проект не рекламирует OAuth, поэтому локальные клиенты вроде LibreChat могут подключаться к нему без ложного запроса авторизации.

## Шаг 2. Создайте app в ChatGPT

В ChatGPT откройте **Settings → Apps → Create**. В workspace Business, Enterprise или Edu администратору может потребоваться сначала включить developer mode и выдать вам право.

Внесите публичный URL MCP endpoint, выберите OAuth authentication, если ChatGPT спросит, и начните tool scan. ChatGPT покажет callback/redirect URL, который будет использовать.

## Шаг 3. Создайте OAuth-пользователя в MCPBox

Не закрывая текущую настройку ChatGPT:

1. Откройте нужный проект MCPBox.
2. Возле информации о точке подключения нажмите кнопку с иконкой пользователей.
3. Введите уникальное имя пользователя и точно вставьте callback URL, показанный ChatGPT.
4. Создайте пользователя, нажмите кнопку показа token и скопируйте его.
5. Вернитесь в то же окно ChatGPT: имя пользователя укажите как OAuth client ID, а token — как OAuth client secret.
6. Оставьте public MCP endpoint как resource/server URL и дождитесь окончания tool scan.

Callback URL сохраняется в базе, но после создания не отображается в таблице. Token также скрыт и показывается только по отдельной кнопке. В таблице видны имя, статус, дата создания и действия управления.

Если ChatGPT сообщает `redirect_uri_mismatch`, его текущий callback отличается от сохранённого. Удалите OAuth-пользователя и создайте его заново с callback из текущего окна ChatGPT. Не начинайте после этого новую настройку app: новый процесс создания может выдать уже другой callback URL.

## Шаг 4. Безопасная проверка

Сначала создайте app как draft. В новом диалоге ChatGPT выберите app через `@` или меню tools и попросите безопасное read-only действие. В **Logs** MCPBox убедитесь, что запрос пришёл. Только после этого разрешайте больше tools или публикуйте app для других пользователей workspace.

## Чек-лист безопасности

- Подключайте только проверенный MCPBox server вашей организации.
- Начинайте с read-only tools и отдельно проверяйте write tools.
- При необходимости создайте отдельный проект для ChatGPT с меньшим набором tools.
- Отключайте или удаляйте отдельного OAuth-пользователя, если его token был раскрыт или сотрудник ушёл.
- После обновления MCP package/server снова проверяйте tools.

Функции ChatGPT apps, доступность тарифов и экраны могут меняться. Актуальные правила workspace смотрите в [Apps in ChatGPT](https://help.openai.com/en/articles/11487775-connectors-in-chatgpt).
