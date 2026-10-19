import { useState } from 'react';
import type { Language } from '../i18n';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from './ui/dialog';

export function SkillDescription({ name, description, language, className = '' }: {
  name: string;
  description: string;
  language: Language;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const label = language === 'ru' ? 'Полное описание' : 'Full description';
  return <div className="min-w-0 flex-1">
    <p className={className}>{description}</p>
    {description && <button
      type="button"
      className="mt-2 text-xs font-medium text-electric-blue underline-offset-4 hover:underline"
      aria-label={`${label}: ${name}`}
      aria-haspopup="dialog"
      onClick={() => setOpen(true)}
    >{label}</button>}
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle className="break-words">{name}</DialogTitle>
          <DialogDescription>{label}</DialogDescription>
        </DialogHeader>
        <p className="whitespace-pre-wrap break-words text-sm leading-relaxed [overflow-wrap:anywhere]">{description}</p>
      </DialogContent>
    </Dialog>
  </div>;
}
