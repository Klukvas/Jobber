import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { stageTemplatesService } from '@/services/stageTemplatesService';
import { Button } from '@/shared/ui/Button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/Card';
import { SkeletonList } from '@/shared/ui/Skeleton';
import { EmptyState } from '@/shared/ui/EmptyState';
import { ErrorState } from '@/shared/ui/ErrorState';
import { Plus, ListOrdered, Trash2, Edit2, Check, Sparkles } from 'lucide-react';
import { CreateStageTemplateModal } from '@/features/stages/modals/CreateStageTemplateModal';
import { usePageTitle } from '@/shared/lib/usePageTitle';

const RECOMMENDED_STAGES = [
  { nameKey: 'stages.applied', order: 0 },
  { nameKey: 'stages.phoneScreen', order: 1 },
  { nameKey: 'stages.technicalInterview', order: 2 },
  { nameKey: 'stages.onsiteInterview', order: 3 },
  { nameKey: 'stages.hrInterview', order: 4 },
  { nameKey: 'stages.offer', order: 5 },
  { nameKey: 'stages.rejected', order: 6 },
];

export default function StageTemplates() {
  usePageTitle('nav.stages');
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['stage-templates'],
    queryFn: () => stageTemplatesService.list({ limit: 100, offset: 0 }),
  });

  const deleteMutation = useMutation({
    mutationFn: stageTemplatesService.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stage-templates'] });
    },
  });

  const createMutation = useMutation({
    mutationFn: stageTemplatesService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stage-templates'] });
    },
  });

  const handleDelete = (id: string, name: string) => {
    if (window.confirm(`Are you sure you want to delete the stage "${name}"?`)) {
      deleteMutation.mutate(id);
    }
  };

  const handleAddRecommended = (nameKey: string, order: number) => {
    createMutation.mutate({ name: t(nameKey), order });
  };

  const handleAddAllRecommended = () => {
    const stages = data?.items || [];
    const existingNames = new Set(stages.map((s) => s.name.toLowerCase()));

    RECOMMENDED_STAGES.forEach((rec) => {
      const name = t(rec.nameKey);
      if (!existingNames.has(name.toLowerCase())) {
        createMutation.mutate({ name, order: rec.order });
      }
    });
  };

  const isRecommendedAdded = (nameKey: string) => {
    const stages = data?.items || [];
    const name = t(nameKey).toLowerCase();
    return stages.some((s) => s.name.toLowerCase() === name);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">{t('stages.title')}</h1>
        </div>
        <SkeletonList count={3} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t('stages.title')}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const stages = data?.items || [];
  const sortedStages = [...stages].sort((a, b) => a.order - b.order);
  const allRecommendedAdded = RECOMMENDED_STAGES.every((rec) => isRecommendedAdded(rec.nameKey));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">{t('stages.title')}</h1>
          <p className="text-muted-foreground mt-1">
            {t('stages.description')}
          </p>
        </div>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          {t('stages.create')}
        </Button>
      </div>

      {/* Recommended Stages */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-primary" />
              <CardTitle>{t('stages.recommended')}</CardTitle>
            </div>
            {!allRecommendedAdded && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleAddAllRecommended}
                disabled={createMutation.isPending}
              >
                <Plus className="h-4 w-4" />
                {t('stages.addAll')}
              </Button>
            )}
          </div>
          <CardDescription>{t('stages.recommendedDescription')}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {RECOMMENDED_STAGES.map((rec) => {
              const added = isRecommendedAdded(rec.nameKey);
              return (
                <Button
                  key={rec.nameKey}
                  variant={added ? 'secondary' : 'outline'}
                  size="sm"
                  disabled={added || createMutation.isPending}
                  onClick={() => handleAddRecommended(rec.nameKey, rec.order)}
                >
                  {added ? (
                    <Check className="h-4 w-4" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  {t(rec.nameKey)}
                </Button>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {/* User's Stage Templates */}
      {sortedStages.length === 0 ? (
        <EmptyState
          icon={<ListOrdered className="h-12 w-12" />}
          title={t('stages.noStages')}
          description={t('stages.noStagesDescription')}
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              {t('stages.create')}
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          {sortedStages.map((stage) => (
            <Card key={stage.id}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">
                    {stage.order}
                  </div>
                  <CardTitle className="text-lg">{stage.name}</CardTitle>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => {
                      // TODO: Implement edit modal
                      alert('Edit functionality coming soon');
                    }}
                  >
                    <Edit2 className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleDelete(stage.id, stage.name)}
                    disabled={deleteMutation.isPending}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  Created {new Date(stage.created_at).toLocaleDateString()}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <CreateStageTemplateModal
        open={isCreateModalOpen}
        onOpenChange={setIsCreateModalOpen}
      />
    </div>
  );
}
