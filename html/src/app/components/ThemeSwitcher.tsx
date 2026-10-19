import { Monitor, Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';
import type { Language } from '../i18n';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuLabel,
  DropdownMenuRadioGroup, DropdownMenuRadioItem, DropdownMenuTrigger,
} from './ui/dropdown-menu';

export function ThemeSwitcher({ language }: { language: Language }) {
  const { theme = 'system', setTheme } = useTheme();
  const copy = language === 'ru'
    ? { label: 'Тема оформления', light: 'Светлая', dark: 'Тёмная', system: 'Системная' }
    : { label: 'Appearance', light: 'Light', dark: 'Dark', system: 'System' };
  const Icon = theme === 'system' ? Monitor : theme === 'dark' ? Moon : Sun;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" aria-label={`${copy.label}: ${copy[theme as 'light' | 'dark' | 'system'] ?? copy.system}`}
          title={copy.label}
          className="inline-flex h-12 w-12 items-center justify-center rounded-2xl border border-border bg-card text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          <Icon className="h-5 w-5" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={8}>
        <DropdownMenuLabel>{copy.label}</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={theme} onValueChange={setTheme}>
          <DropdownMenuRadioItem value="light"><Sun className="mr-2 h-4 w-4" />{copy.light}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="dark"><Moon className="mr-2 h-4 w-4" />{copy.dark}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="system"><Monitor className="mr-2 h-4 w-4" />{copy.system}</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
