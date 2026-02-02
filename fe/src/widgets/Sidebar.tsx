import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useSidebarStore } from '@/stores/sidebarStore';
import { cn } from '@/shared/lib/utils';
import {
  Briefcase,
  FileText,
  Building2,
  Search,
  ListOrdered,
  Settings,
  ChevronLeft,
  ChevronRight,
  X,
} from 'lucide-react';

const navItems = [
  { path: '/app/applications', icon: Briefcase, labelKey: 'nav.applications' },
  { path: '/app/resumes', icon: FileText, labelKey: 'nav.resumes' },
  { path: '/app/companies', icon: Building2, labelKey: 'nav.companies' },
  { path: '/app/jobs', icon: Search, labelKey: 'nav.jobs' },
  { path: '/app/stages', icon: ListOrdered, labelKey: 'nav.stages' },
  { path: '/app/settings', icon: Settings, labelKey: 'nav.settings' },
];

export function Sidebar() {
  const { t } = useTranslation();
  const { isExpanded, isMobileOpen, toggleExpanded, toggleMobile, closeMobile } =
    useSidebarStore();

  return (
    <>
      {/* Mobile Overlay */}
      {isMobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-background/80 backdrop-blur-sm md:hidden"
          onClick={closeMobile}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed left-0 top-0 z-50 h-screen border-r bg-card transition-all duration-300',
          'md:relative md:z-0',
          {
            'w-64': isExpanded,
            'w-16': !isExpanded,
            '-translate-x-full md:translate-x-0': !isMobileOpen,
            'translate-x-0': isMobileOpen,
          }
        )}
      >
        <div className="flex h-full flex-col">
          {/* Logo / Brand */}
          <div className="flex h-16 items-center justify-between border-b px-4">
            {isExpanded && (
              <h1 className="text-xl font-bold">Jobber</h1>
            )}
            <button
              onClick={toggleExpanded}
              className="hidden rounded-md p-2 hover:bg-accent md:block"
              aria-label={isExpanded ? 'Collapse sidebar' : 'Expand sidebar'}
            >
              {isExpanded ? (
                <ChevronLeft className="h-5 w-5" />
              ) : (
                <ChevronRight className="h-5 w-5" />
              )}
            </button>
            <button
              onClick={closeMobile}
              className="rounded-md p-2 hover:bg-accent md:hidden"
              aria-label="Close sidebar"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 p-2">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.path}
                  to={item.path}
                  onClick={() => {
                    if (window.innerWidth < 768) {
                      closeMobile();
                    }
                  }}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      'hover:bg-accent hover:text-accent-foreground',
                      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                      {
                        'bg-accent text-accent-foreground': isActive,
                        'text-muted-foreground': !isActive,
                        'justify-center': !isExpanded,
                      }
                    )
                  }
                  title={!isExpanded ? t(item.labelKey) : undefined}
                >
                  <Icon className="h-5 w-5 flex-shrink-0" />
                  {isExpanded && <span>{t(item.labelKey)}</span>}
                </NavLink>
              );
            })}
          </nav>
        </div>
      </aside>

      {/* Mobile Menu Button */}
      <button
        onClick={toggleMobile}
        className="fixed bottom-4 right-4 z-40 rounded-full bg-primary p-4 text-primary-foreground shadow-lg md:hidden"
        aria-label="Open menu"
      >
        <Briefcase className="h-6 w-6" />
      </button>
    </>
  );
}
