export type Language = 'en' | 'ru';

type Dictionary = {
  labels: {
    appTitle: string;
    controlCenter: string;
    appDescription: string;
    projects: string;
    createProject: string;
    name: string;
    description: string;
    projectOverview: string;
    servers: string;
    running: string;
    connectionEndpoint: string;
    addServer: string;
    serverName: string;
    launchCommand: string;
    stdio: string;
    httpStreaming: string;
    workingDirectory: string;
    url: string;
    autoStart: string;
    copied: string;
    copyUrl: string;
    noProjectSelected: string;
    ready: string;
    notSelected: string;
    notSpecified: string;
    stop: string;
    start: string;
    language: string;
    english: string;
    russian: string;
    arguments: string;
    environmentVariables: string;
    environmentVariablePassthrough: string;
    bearerTokenEnvironmentVariable: string;
    headers: string;
    headersFromEnvironmentVariables: string;
    addArgument: string;
    addEnvironmentVariable: string;
    addHeader: string;
    addVariable: string;
    key: string;
    value: string;
    logs: string;
    auditLogs: string;
    refresh: string;
    pauseProject: string;
    resumeProject: string;
    paused: string;
    disabled: string;
    disableServer: string;
    enableServer: string;
    remote: string;
    unknownActor: string;
    requestControl: string;
    filterByProject: string;
    allProjects: string;
    currentProjectOnly: string;
    activityOverview: string;
    hottestProject: string;
    hottestServer: string;
    requests: string;
    noActivity: string;
    consoleFeed: string;
    info: string;
    serverInfo: string;
    capabilities: string;
    tools: string;
    resources: string;
    prompts: string;
    readme: string;
    instructions: string;
    protocolVersion: string;
    version: string;
    noReadme: string;
    noTools: string;
    noResources: string;
    noPrompts: string;
    inspectServer: string;
    health: string;
    healthy: string;
    failed: string;
    unknown: string;
    lastCheck: string;
    check: string;
    oauth: string;
    connected: string;
    notConnected: string;
    oauthConnected: string;
    market: string;
    integrations: string;
    catalog: string;
    catalogItems: string;
    installed: string;
    lastSync: string;
    externalManifestUrl: string;
    syncCatalog: string;
    allCategories: string;
    command: string;
    endpoint: string;
    docs: string;
    website: string;
    generalCategory: string;
    mcpDiscovery: string;
    installPackage: string;
    packageInstalled: string;
    packageNotInstalled: string;
    addToProject: string;
    addedToProject: string;
    notInProject: string;
  };
  messages: {
    loadingProjects: string;
    noProjects: string;
    projectHelper: string;
    projectNamePlaceholder: string;
    projectDescriptionPlaceholder: string;
    emptySelection: string;
    emptySelectionBody: string;
    overviewFallbackDescription: string;
    connectionDescription: string;
    connectionWarning: (token: string) => string;
    addServerDescription: string;
    serverNamePlaceholder: string;
    launchCommandPlaceholder: string;
    commandPlaceholder: string;
    argumentPlaceholder: string;
    workingDirectoryPlaceholder: string;
    urlPlaceholder: string;
    bearerTokenPlaceholder: string;
    envPassthroughPlaceholder: string;
    serverControlDescription: string;
    noServers: string;
    workspaceGroupFallback: string;
    serverCount: (count: number) => string;
    runningCount: (count: number) => string;
    createProjectError: string;
    loadProjectsError: string;
    addServerError: string;
    startServerError: string;
    stopServerError: string;
    setProjectPausedError: string;
    setServerEnabledError: string;
    logsDescription: string;
    loadingLogs: string;
    noLogs: string;
    projectTag: (id: number) => string;
    serverTag: (id: number) => string;
    consoleDescription: string;
    popularityDescription: string;
    inspectDescription: string;
    inspectServerError: string;
    checkServerError: string;
    requestFailed: (status: number) => string;
    marketDescription: string;
    notSynced: string;
    advancedModeEnabled: string;
    selectProjectBeforeInstall: string;
    noDescriptionProvided: string;
    upstreamAuthNotice: string;
    noValue: string;
    workingDirectoryValue: (path: string) => string;
    autoStartAfterInstall: string;
    syncManifestToPopulateCatalog: string;
    noIntegrationsInCategory: string;
    installIntegrationTitle: (name: string) => string;
    installIntegrationFallbackTitle: string;
    installIntegrationDescription: string;
    oneValuePerLine: string;
    installIntegrationAction: string;
    loadPackagesError: string;
    installPackageError: string;
    addPackageToProjectError: string;
    addPackageDialogTitle: (name: string) => string;
    addPackageDialogFallbackTitle: string;
    addPackageDialogDescription: string;
  };
};

export const defaultLanguage: Language = 'en';
export const languageStorageKey = 'mcpbox.language';

export const dictionaries: Record<Language, Dictionary> = {
  en: {
    labels: {
      appTitle: 'MCPBox',
      controlCenter: '',
      appDescription: 'Unified service for managing MCP servers.',
      projects: 'Projects',
      createProject: 'Create Project',
      name: 'Name',
      description: 'Description',
      projectOverview: 'Project Overview',
      servers: 'Servers',
      running: 'Running',
      connectionEndpoint: 'Connection Endpoint',
      addServer: 'Add MCP Server',
      serverName: 'Server name',
      launchCommand: 'Launch command',
      stdio: 'STDIO',
      httpStreaming: 'HTTP streaming',
      workingDirectory: 'Working directory',
      url: 'URL',
      autoStart: 'Start automatically when MCPBox boots',
      copied: 'Copied',
      copyUrl: 'Copy URL',
      noProjectSelected: 'No project selected',
      ready: 'Ready',
      notSelected: 'Not selected',
      notSpecified: 'Not specified',
      stop: 'Stop',
      start: 'Start',
      language: 'Language',
      english: 'English',
      russian: 'Russian',
      arguments: 'Arguments',
      environmentVariables: 'Environment variables',
      environmentVariablePassthrough: 'Environment variable passthrough',
      bearerTokenEnvironmentVariable: 'Bearer token environment variable',
      headers: 'Headers',
      headersFromEnvironmentVariables: 'Headers from environment variables',
      addArgument: 'Add argument',
      addEnvironmentVariable: 'Add environment variable',
      addHeader: 'Add header',
      addVariable: 'Add variable',
      key: 'Key',
      value: 'Value',
      logs: 'Logs',
      auditLogs: 'Audit Logs',
      refresh: 'Refresh',
      pauseProject: 'Pause project',
      resumeProject: 'Resume project',
      paused: 'Paused',
      disabled: 'Disabled',
      disableServer: 'Disable server',
      enableServer: 'Enable server',
      remote: 'Remote',
      unknownActor: 'unknown actor',
      requestControl: 'Request control',
      filterByProject: 'Filter by project',
      allProjects: 'All projects',
      currentProjectOnly: 'Current project only',
      activityOverview: 'Activity overview',
      hottestProject: 'Hottest project',
      hottestServer: 'Hottest server',
      requests: 'requests',
      noActivity: 'No activity yet',
      consoleFeed: 'Console feed',
      info: 'Info',
      serverInfo: 'Server Info',
      capabilities: 'Capabilities',
      tools: 'Tools',
      resources: 'Resources',
      prompts: 'Prompts',
      readme: 'README',
      instructions: 'Instructions',
      protocolVersion: 'Protocol version',
      version: 'Version',
      noReadme: 'README not found near the local server path.',
      noTools: 'No tools exposed.',
      noResources: 'No resources exposed.',
      noPrompts: 'No prompts exposed.',
      inspectServer: 'Inspect server',
      health: 'Health',
      healthy: 'Healthy',
      failed: 'Failed',
      unknown: 'Unknown',
      lastCheck: 'Last check',
      check: 'Check',
      oauth: 'OAuth',
      connected: 'Connected',
      notConnected: 'Not connected',
      oauthConnected: 'OAuth connected',
      market: 'Market',
      integrations: 'Integrations',
      catalog: 'Catalog',
      catalogItems: 'Catalog items',
      installed: 'Installed',
      lastSync: 'Last sync',
      externalManifestUrl: 'External manifest URL',
      syncCatalog: 'Sync catalog',
      allCategories: 'All categories',
      command: 'Command',
      endpoint: 'Endpoint',
      docs: 'Docs',
      website: 'Website',
      generalCategory: 'general',
      mcpDiscovery: 'mcp discovery',
      installPackage: 'Install package',
      packageInstalled: 'Package installed',
      packageNotInstalled: 'Package not installed',
      addToProject: 'Add to project',
      addedToProject: 'Added to project',
      notInProject: 'Not in project',
    },
    messages: {
      loadingProjects: 'Loading projects...',
      noProjects: 'No projects yet. Create the first workspace below.',
      projectHelper: 'A project is a logical group of MCP servers for one environment.',
      projectNamePlaceholder: 'Client Workspace',
      projectDescriptionPlaceholder: 'What this workspace group is for',
      emptySelection:
        'Create the first workspace on the left and it will become the control center for its MCP servers.',
      emptySelectionBody:
        'Create the first workspace on the left and it will become the control center for its MCP servers.',
      overviewFallbackDescription: 'Workspace group for MCP clients and tools.',
      connectionDescription:
        'All clients in this workspace connect through the project token and can access all enabled MCP servers in the project.',
      connectionWarning: (token: string) => `To use /mcp/${token}, add and enable at least one MCP server.`,
      addServerDescription:
        'Add one or more MCP servers to expose them through the shared project endpoint.',
      serverNamePlaceholder: 'Filesystem Server',
      launchCommandPlaceholder: 'npx -y @modelcontextprotocol/server-filesystem /path',
      commandPlaceholder: 'uvx mcp-server or node dist/index.js',
      argumentPlaceholder: '--port or ./path',
      workingDirectoryPlaceholder: '/absolute/path or leave empty',
      urlPlaceholder: 'https://mcp.example.com/mcp',
      bearerTokenPlaceholder: 'MCP_BEARER_TOKEN',
      envPassthroughPlaceholder: 'OPENAI_API_KEY',
      serverControlDescription:
        'Manage the processes and availability of all MCP servers exposed through the project endpoint.',
      noServers: 'No servers have been added to this project yet.',
      workspaceGroupFallback: 'Workspace group for MCP clients',
      serverCount: (count: number) => `${count} servers`,
      runningCount: (count: number) => `${count} running`,
      createProjectError: 'Failed to create project',
      loadProjectsError: 'Failed to load projects',
      addServerError: 'Failed to add server',
      startServerError: 'Failed to start server',
      stopServerError: 'Failed to stop server',
      setProjectPausedError: 'Failed to update project state',
      setServerEnabledError: 'Failed to update server state',
      logsDescription: 'Request monitoring and control actions across projects and MCP servers.',
      loadingLogs: 'Loading logs...',
      noLogs: 'No audit logs yet.',
      projectTag: (id: number) => `project #${id}`,
      serverTag: (id: number) => `server #${id}`,
      consoleDescription: 'Compact event stream for requests, connects, and control actions.',
      popularityDescription:
        'Who is receiving the most MCP traffic right now based on the current filter.',
      inspectDescription:
        'Live MCP inspection for this local STDIO server plus nearby README if found.',
      inspectServerError: 'Failed to inspect server',
      checkServerError: 'Failed to verify server health',
      requestFailed: (status: number) => `Request failed with status ${status}`,
      marketDescription:
        'Sync the external integration manifest into SQLite and install selected items into the current project as linked MCP servers.',
      notSynced: 'Not synced',
      advancedModeEnabled: 'Advanced mode enabled. Press Cmd/Ctrl + Shift + U to hide.',
      selectProjectBeforeInstall:
        'Create or select a project before installing integrations.',
      noDescriptionProvided: 'No description provided.',
      upstreamAuthNotice:
        'Authentication is handled by the upstream MCP server. After install, your MCP client should complete the sign-in flow when it connects through MCPBox.',
      noValue: 'n/a',
      workingDirectoryValue: (path: string) => `Working directory: ${path}`,
      autoStartAfterInstall: 'Starts automatically after install',
      syncManifestToPopulateCatalog: 'Sync the external manifest to populate the catalog.',
      noIntegrationsInCategory: 'No integrations in this category yet.',
      installIntegrationTitle: (name: string) => `Install ${name}`,
      installIntegrationFallbackTitle: 'Install integration',
      installIntegrationDescription:
        'Fill in the required connection settings before adding this integration to the selected project.',
      oneValuePerLine: 'One value per line',
      installIntegrationAction: 'Install integration',
      loadPackagesError: 'Failed to load packages',
      installPackageError: 'Failed to install package',
      addPackageToProjectError: 'Failed to add package to project',
      addPackageDialogTitle: (name: string) => `Add ${name} to project`,
      addPackageDialogFallbackTitle: 'Add package to project',
      addPackageDialogDescription:
        'Choose a project and configure the selected package instance before adding it.',
    },
  },
  ru: {
    labels: {
      appTitle: 'MCPBox',
      controlCenter: '',
      appDescription: 'Единый сервис для управления MCP-серверами.',
      projects: 'Проекты',
      createProject: 'Создать проект',
      name: 'Название',
      description: 'Описание',
      projectOverview: 'Обзор проекта',
      servers: 'Серверы',
      running: 'Запущено',
      connectionEndpoint: 'Точка подключения',
      addServer: 'Добавить MCP-сервер',
      serverName: 'Название сервера',
      launchCommand: 'Команда запуска',
      stdio: 'STDIO',
      httpStreaming: 'Потоковая передача HTTP',
      workingDirectory: 'Рабочая директория',
      url: 'URL',
      autoStart: 'Запускать автоматически при старте MCPBox',
      copied: 'Скопировано',
      copyUrl: 'Скопировать URL',
      noProjectSelected: 'Проект не выбран',
      ready: 'Готово',
      notSelected: 'Не выбран',
      notSpecified: 'Не указано',
      stop: 'Остановить',
      start: 'Запустить',
      language: 'Язык',
      english: 'English',
      russian: 'Русский',
      arguments: 'Аргументы',
      environmentVariables: 'Переменные окружения',
      environmentVariablePassthrough: 'Передача переменных окружения',
      bearerTokenEnvironmentVariable: 'Переменная окружения Bearer-токена',
      headers: 'Заголовки',
      headersFromEnvironmentVariables: 'Заголовки из переменных окружения',
      addArgument: 'Добавить аргумент',
      addEnvironmentVariable: 'Добавить переменную окружения',
      addHeader: 'Добавить заголовок',
      addVariable: 'Добавить переменную',
      key: 'Ключ',
      value: 'Значение',
      logs: 'Логи',
      auditLogs: 'Журнал аудита',
      refresh: 'Обновить',
      pauseProject: 'Приостановить проект',
      resumeProject: 'Возобновить проект',
      paused: 'Приостановлен',
      disabled: 'Отключен',
      disableServer: 'Отключить сервер',
      enableServer: 'Включить сервер',
      remote: 'Удаленный',
      unknownActor: 'неизвестный источник',
      requestControl: 'Контроль запросов',
      filterByProject: 'Фильтр по проекту',
      allProjects: 'Все проекты',
      currentProjectOnly: 'Только текущий проект',
      activityOverview: 'Сводка активности',
      hottestProject: 'Самый активный проект',
      hottestServer: 'Самый активный сервер',
      requests: 'запросов',
      noActivity: 'Активности пока нет',
      consoleFeed: 'Поток событий',
      info: 'Инфо',
      serverInfo: 'Информация о сервере',
      capabilities: 'Возможности',
      tools: 'Инструменты',
      resources: 'Ресурсы',
      prompts: 'Промпты',
      readme: 'README',
      instructions: 'Инструкции',
      protocolVersion: 'Версия протокола',
      version: 'Версия',
      noReadme: 'README рядом с локальным сервером не найден.',
      noTools: 'Инструменты не найдены.',
      noResources: 'Ресурсы не найдены.',
      noPrompts: 'Промпты не найдены.',
      inspectServer: 'Проверить сервер',
      health: 'Состояние',
      healthy: 'Исправен',
      failed: 'Ошибка',
      unknown: 'Неизвестно',
      lastCheck: 'Последняя проверка',
      check: 'Проверить',
      oauth: 'OAuth',
      connected: 'Подключен',
      notConnected: 'Не подключен',
      oauthConnected: 'OAuth подключен',
      market: 'Маркет',
      integrations: 'Интеграции',
      catalog: 'Каталог',
      catalogItems: 'Элементы каталога',
      installed: 'Установлено',
      lastSync: 'Последняя синхронизация',
      externalManifestUrl: 'URL внешнего манифеста',
      syncCatalog: 'Синхронизировать каталог',
      allCategories: 'Все категории',
      command: 'Команда',
      endpoint: 'Endpoint',
      docs: 'Документация',
      website: 'Сайт',
      generalCategory: 'общее',
      mcpDiscovery: 'mcp discovery',
      installPackage: 'Установить пакет',
      packageInstalled: 'Пакет установлен',
      packageNotInstalled: 'Пакет не установлен',
      addToProject: 'Добавить в проект',
      addedToProject: 'Добавлено в проект',
      notInProject: 'Не в проекте',
    },
    messages: {
      loadingProjects: 'Загружаю проекты...',
      noProjects: 'Проектов пока нет. Создай первый workspace ниже.',
      projectHelper: 'Проект это логическая группа MCP-серверов для одного окружения.',
      projectNamePlaceholder: 'Клиентский workspace',
      projectDescriptionPlaceholder: 'Для чего нужен этот workspace',
      emptySelection:
        'Создай первый workspace слева, и он станет центром управления своими MCP-серверами.',
      emptySelectionBody:
        'Создай первый workspace слева, и он станет центром управления своими MCP-серверами.',
      overviewFallbackDescription: 'Логическая группа для MCP-клиентов и инструментов.',
      connectionDescription:
        'Все клиенты этого workspace подключаются через project token и получают доступ ко всем включённым MCP-серверам проекта.',
      connectionWarning: (token: string) => `Чтобы использовать /mcp/${token}, добавьте и включите хотя бы один MCP-сервер.`,
      addServerDescription:
        'Добавьте один или несколько MCP-серверов, чтобы опубликовать их через общий endpoint проекта.',
      serverNamePlaceholder: 'Filesystem Server',
      launchCommandPlaceholder: 'npx -y @modelcontextprotocol/server-filesystem /path',
      commandPlaceholder: 'uvx mcp-server или node dist/index.js',
      argumentPlaceholder: '--port или ./path',
      workingDirectoryPlaceholder: '/absolute/path или оставить пустым',
      urlPlaceholder: 'https://mcp.example.com/mcp',
      bearerTokenPlaceholder: 'MCP_BEARER_TOKEN',
      envPassthroughPlaceholder: 'OPENAI_API_KEY',
      serverControlDescription:
        'Управляй процессами и доступностью всех MCP-серверов, опубликованных через endpoint проекта.',
      noServers: 'В этом проекте пока нет серверов.',
      workspaceGroupFallback: 'Логическая группа для MCP-клиентов',
      serverCount: (count: number) => `${count} серверов`,
      runningCount: (count: number) => `${count} запущено`,
      createProjectError: 'Не удалось создать проект',
      loadProjectsError: 'Не удалось загрузить проекты',
      addServerError: 'Не удалось добавить сервер',
      startServerError: 'Не удалось запустить сервер',
      stopServerError: 'Не удалось остановить сервер',
      setProjectPausedError: 'Не удалось обновить состояние проекта',
      setServerEnabledError: 'Не удалось обновить состояние сервера',
      logsDescription: 'Мониторинг запросов и действий управления по проектам и MCP-серверам.',
      loadingLogs: 'Загружаю логи...',
      noLogs: 'Записей аудита пока нет.',
      projectTag: (id: number) => `проект #${id}`,
      serverTag: (id: number) => `сервер #${id}`,
      consoleDescription:
        'Компактный поток событий по запросам, подключениям и управляющим действиям.',
      popularityDescription:
        'Кто сейчас получает больше всего MCP-трафика в рамках текущего фильтра.',
      inspectDescription:
        'Живой MCP inspection для локального STDIO сервера и nearby README, если он найден.',
      inspectServerError: 'Не удалось проинспектировать сервер',
      checkServerError: 'Не удалось проверить сервер',
      requestFailed: (status: number) => `Запрос завершился со статусом ${status}`,
      marketDescription:
        'Синхронизируйте внешний манифест интеграций в SQLite и устанавливайте выбранные элементы в текущий проект как связанные MCP-серверы.',
      notSynced: 'Не синхронизировано',
      advancedModeEnabled:
        'Расширенный режим включён. Нажмите Cmd/Ctrl + Shift + U, чтобы скрыть поле.',
      selectProjectBeforeInstall:
        'Выберите проект в боковой панели перед установкой интеграций.',
      noDescriptionProvided: 'Описание отсутствует.',
      upstreamAuthNotice:
        'Аутентификация выполняется на стороне исходного MCP-сервера. После установки MCP-клиент должен завершить вход при подключении через MCPBox.',
      noValue: 'n/a',
      workingDirectoryValue: (path: string) => `Рабочая директория: ${path}`,
      autoStartAfterInstall: 'Запускается автоматически после установки',
      syncManifestToPopulateCatalog:
        'Синхронизируйте внешний манифест, чтобы заполнить каталог.',
      noIntegrationsInCategory: 'В этой категории пока нет интеграций.',
      installIntegrationTitle: (name: string) => `Установить ${name}`,
      installIntegrationFallbackTitle: 'Установка интеграции',
      installIntegrationDescription:
        'Заполните обязательные параметры подключения перед добавлением этой интеграции в выбранный проект.',
      oneValuePerLine: 'По одному значению на строку',
      installIntegrationAction: 'Установить',
      loadPackagesError: 'Не удалось загрузить пакеты',
      installPackageError: 'Не удалось установить пакет',
      addPackageToProjectError: 'Не удалось добавить пакет в проект',
      addPackageDialogTitle: (name: string) => `Добавить ${name} в проект`,
      addPackageDialogFallbackTitle: 'Добавить пакет в проект',
      addPackageDialogDescription:
        'Настройте выбранный экземпляр пакета перед добавлением в текущий проект.',
    },
  },
};

export function detectInitialLanguage(): Language {
  if (typeof window === 'undefined') {
    return defaultLanguage;
  }

  const storedLanguage = window.localStorage.getItem(languageStorageKey);
  if (storedLanguage === 'en' || storedLanguage === 'ru') {
    return storedLanguage;
  }

  return defaultLanguage;
}
