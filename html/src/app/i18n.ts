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
    primary: string;
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
    primaryRequired: string;
    notSelected: string;
    notSpecified: string;
    primaryServer: string;
    setAsPrimary: string;
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
    updatePrimaryError: string;
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
    requestFailed: (status: number) => string;
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
      primary: 'Primary',
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
      primaryRequired: 'Primary server required',
      notSelected: 'Not selected',
      notSpecified: 'Not specified',
      primaryServer: 'Primary server',
      setAsPrimary: 'Set as primary',
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
      noReadme: 'README not found рядом with the local server path.',
      noTools: 'No tools exposed.',
      noResources: 'No resources exposed.',
      noPrompts: 'No prompts exposed.',
      inspectServer: 'Inspect server',
    },
    messages: {
      loadingProjects: 'Loading projects...',
      noProjects: 'No projects yet. Create the first workspace below.',
      projectHelper: 'A project is a logical group of MCP servers for one environment.',
      projectNamePlaceholder: 'Client Workspace',
      projectDescriptionPlaceholder: 'What this workspace group is for',
      emptySelection: 'Create the first workspace on the left and it will become the control center for its MCP servers.',
      emptySelectionBody: 'Create the first workspace on the left and it will become the control center for its MCP servers.',
      overviewFallbackDescription: 'Workspace group for MCP clients and tools.',
      connectionDescription: 'All clients in this workspace connect through the project token and the primary server.',
      connectionWarning: (token: string) =>
        `To use /connect/${token}, assign a primary server first.`,
      addServerDescription:
        'The first server becomes primary automatically. Additional servers stay in the same workspace.',
      serverNamePlaceholder: 'Filesystem Server',
      launchCommandPlaceholder: 'npx -y @modelcontextprotocol/server-filesystem /path',
      commandPlaceholder: 'uvx mcp-server or node dist/index.js',
      argumentPlaceholder: '--port or ./path',
      workingDirectoryPlaceholder: '/absolute/path or leave empty',
      urlPlaceholder: 'https://mcp.example.com/mcp',
      bearerTokenPlaceholder: 'MCP_BEARER_TOKEN',
      envPassthroughPlaceholder: 'OPENAI_API_KEY',
      serverControlDescription:
        'Manage processes and choose the primary server for the connect endpoint.',
      noServers: 'No servers have been added to this project yet.',
      workspaceGroupFallback: 'Workspace group for MCP clients',
      serverCount: (count: number) => `${count} servers`,
      runningCount: (count: number) => `${count} running`,
      createProjectError: 'Failed to create project',
      loadProjectsError: 'Failed to load projects',
      addServerError: 'Failed to add server',
      updatePrimaryError: 'Failed to update primary server',
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
      popularityDescription: 'Who is receiving the most MCP traffic right now based on the current filter.',
      inspectDescription: 'Live MCP inspection for this local STDIO server plus nearby README if found.',
      inspectServerError: 'Failed to inspect server',
      requestFailed: (status: number) => `Request failed with status ${status}`,
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
      primary: 'Основной',
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
      primaryRequired: 'Нужен primary server',
      notSelected: 'Не выбран',
      notSpecified: 'Не указана',
      primaryServer: 'Primary server',
      setAsPrimary: 'Сделать primary',
      stop: 'Остановить',
      start: 'Запустить',
      language: 'Язык',
      english: 'English',
      russian: 'Русский',
      arguments: 'Аргументы',
      environmentVariables: 'Переменные окружения',
      environmentVariablePassthrough: 'Передача переменных окружения',
      bearerTokenEnvironmentVariable: 'Переменная окружения токена Bearer',
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
      tools: 'Tools',
      resources: 'Resources',
      prompts: 'Prompts',
      readme: 'README',
      instructions: 'Инструкции',
      protocolVersion: 'Версия протокола',
      version: 'Версия',
      noReadme: 'README рядом с локальным сервером не найден.',
      noTools: 'Tools не найдены.',
      noResources: 'Resources не найдены.',
      noPrompts: 'Prompts не найдены.',
      inspectServer: 'Проверить сервер',
    },
    messages: {
      loadingProjects: 'Загружаю проекты...',
      noProjects: 'Проектов пока нет. Создай первый workspace ниже.',
      projectHelper: 'Проект — это логическая группа MCP-серверов для одного окружения.',
      projectNamePlaceholder: 'Клиентский workspace',
      projectDescriptionPlaceholder: 'Для чего нужен этот workspace',
      emptySelection: 'Создай первый workspace слева, и он станет центром управления своими MCP-серверами.',
      emptySelectionBody: 'Создай первый workspace слева, и он станет центром управления своими MCP-серверами.',
      overviewFallbackDescription: 'Логическая группа для MCP-клиентов и инструментов.',
      connectionDescription: 'Все клиенты этого workspace подключаются через project token и primary server.',
      connectionWarning: (token: string) =>
        `Чтобы использовать /connect/${token}, сначала назначь primary server.`,
      addServerDescription:
        'Первый сервер автоматически становится primary. Остальные остаются в том же workspace.',
      serverNamePlaceholder: 'Filesystem Server',
      launchCommandPlaceholder: 'npx -y @modelcontextprotocol/server-filesystem /path',
      commandPlaceholder: 'uvx mcp-server или node dist/index.js',
      argumentPlaceholder: '--port или ./path',
      workingDirectoryPlaceholder: '/absolute/path или оставить пустым',
      urlPlaceholder: 'https://mcp.example.com/mcp',
      bearerTokenPlaceholder: 'MCP_BEARER_TOKEN',
      envPassthroughPlaceholder: 'OPENAI_API_KEY',
      serverControlDescription:
        'Управляй процессами и выбирай primary server для connect endpoint.',
      noServers: 'В этом проекте пока нет серверов.',
      workspaceGroupFallback: 'Логическая группа для MCP-клиентов',
      serverCount: (count: number) => `${count} серверов`,
      runningCount: (count: number) => `${count} запущено`,
      createProjectError: 'Не удалось создать проект',
      loadProjectsError: 'Не удалось загрузить проекты',
      addServerError: 'Не удалось добавить сервер',
      updatePrimaryError: 'Не удалось обновить primary server',
      startServerError: 'Не удалось запустить сервер',
      stopServerError: 'Не удалось остановить сервер',
      setProjectPausedError: 'Не удалось обновить состояние проекта',
      setServerEnabledError: 'Не удалось обновить состояние сервера',
      logsDescription: 'Мониторинг запросов и действий управления по проектам и MCP-серверам.',
      loadingLogs: 'Загружаю логи...',
      noLogs: 'Записей аудита пока нет.',
      projectTag: (id: number) => `проект #${id}`,
      serverTag: (id: number) => `сервер #${id}`,
      consoleDescription: 'Компактный поток событий по запросам, подключениям и управляющим действиям.',
      popularityDescription: 'Кто сейчас получает больше всего MCP-трафика в рамках текущего фильтра.',
      inspectDescription: 'Живой MCP inspection для локального STDIO сервера и nearby README, если он найден.',
      inspectServerError: 'Не удалось проинспектировать сервер',
      requestFailed: (status: number) => `Запрос завершился со статусом ${status}`,
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
