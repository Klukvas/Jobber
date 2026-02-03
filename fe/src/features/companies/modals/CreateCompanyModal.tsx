import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { companiesService } from '@/services/companiesService';
import type { CompanyDTO, UpdateCompanyRequest } from '@/shared/types/api';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/shared/ui/Dialog';
import { Button } from '@/shared/ui/Button';
import { Input } from '@/shared/ui/Input';
import { Label } from '@/shared/ui/Label';

interface CreateCompanyModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  company?: CompanyDTO; // If provided, modal is in edit mode
}

// Inner content component that resets state when key changes
function ModalContent({ company, onOpenChange }: Omit<CreateCompanyModalProps, 'open'>) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  
  const [name, setName] = useState(company?.name || '');
  const [location, setLocation] = useState(company?.location || '');
  const [notes, setNotes] = useState(company?.notes || '');

  const isEditMode = !!company;

  const createMutation = useMutation({
    mutationFn: companiesService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] });
      onOpenChange(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateCompanyRequest }) =>
      companiesService.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] });
      onOpenChange(false);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name) {
      if (isEditMode && company) {
        updateMutation.mutate({
          id: company.id,
          data: {
            name,
            location: location || undefined,
            notes: notes || undefined,
          },
        });
      } else {
        createMutation.mutate({
          name,
          location: location || undefined,
          notes: notes || undefined,
        });
      }
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <>
      <DialogHeader>
        <DialogTitle>
          {isEditMode ? t('companies.edit') : t('companies.create')}
        </DialogTitle>
        <DialogDescription>
          {isEditMode
            ? 'Update company information'
            : 'Add a new company to your database'}
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit}>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('companies.name')} *</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Google, Microsoft, Startup Inc."
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="location">{t('companies.location')}</Label>
            <Input
              id="location"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="e.g., San Francisco, CA"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="notes">{t('companies.notes')}</Label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Any additional notes about the company..."
              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            {t('common.cancel')}
          </Button>
          <Button type="submit" disabled={isPending || !name}>
            {isPending
              ? t('common.loading')
              : isEditMode
              ? t('common.save')
              : t('common.create')}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}

export function CreateCompanyModal({ open, onOpenChange, company }: CreateCompanyModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        {/* Key prop resets the form state when company changes or modal reopens */}
        <ModalContent 
          key={`${company?.id || 'new'}-${open}`}
          company={company} 
          onOpenChange={onOpenChange} 
        />
      </DialogContent>
    </Dialog>
  );
}
