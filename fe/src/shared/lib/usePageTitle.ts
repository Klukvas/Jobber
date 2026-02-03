import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

export function usePageTitle(titleKey?: string) {
  const { t } = useTranslation();

  useEffect(() => {
    if (titleKey) {
      document.title = `${t(titleKey)} - Jobber`;
    } else {
      document.title = 'Jobber';
    }
  }, [titleKey, t]);
}
