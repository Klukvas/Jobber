import { useTranslation } from 'react-i18next';
import { useThemeStore } from '@/stores/themeStore';
import { Button } from '@/shared/ui/Button';
import { Sun, Moon, Languages } from 'lucide-react';
import { cn } from '@/shared/lib/utils';
import { useState } from 'react';

export function Header() {
  const { i18n } = useTranslation();
  const { theme, toggleTheme } = useThemeStore();
  const [showLanguageMenu, setShowLanguageMenu] = useState(false);

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    setShowLanguageMenu(false);
  };

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-end gap-2 border-b bg-background px-4">
      {/* Theme Toggle */}
      <Button
        variant="ghost"
        size="icon"
        onClick={toggleTheme}
        aria-label={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
      >
        {theme === 'light' ? (
          <Sun className="h-5 w-5" />
        ) : (
          <Moon className="h-5 w-5" />
        )}
      </Button>

      {/* Language Switcher */}
      <div className="relative">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setShowLanguageMenu(!showLanguageMenu)}
          aria-label="Change language"
          aria-expanded={showLanguageMenu}
        >
          <Languages className="h-5 w-5" />
        </Button>

        {showLanguageMenu && (
          <>
            <div
              className="fixed inset-0 z-40"
              onClick={() => setShowLanguageMenu(false)}
            />
            <div className="absolute right-0 top-full z-50 mt-2 w-32 rounded-md border bg-popover p-1 shadow-md">
              <button
                onClick={() => changeLanguage('en')}
                className={cn(
                  'w-full rounded-sm px-3 py-2 text-left text-sm hover:bg-accent',
                  i18n.language === 'en' && 'bg-accent'
                )}
              >
                English
              </button>
              {/* Add more languages as needed */}
            </div>
          </>
        )}
      </div>
    </header>
  );
}
