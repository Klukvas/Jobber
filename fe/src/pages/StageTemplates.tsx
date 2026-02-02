import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { stageTemplatesService } from '@/services/stageTemplatesService';
import { Button } from '@/shared/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/Card';
import { SkeletonList } from '@/shared/ui/Skeleton';
import { EmptyState } from '@/shared/ui/EmptyState';
import { ErrorState } from '@/shared/ui/ErrorState';
import { Plus, ListOrdered, Trash2, Edit2 } from 'lucide-react';
import { CreateStageTemplateModal } from '@/features/stages/modals/CreateStageTemplateModal';

export default function StageTemplates() {
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

  const handleDelete = (id: string, name: string) => {
    if (window.confirm(`Are you sure you want to delete the stage "${name}"?`)) {
      deleteMutation.mutate(id);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">Stage Templates</h1>
        </div>
        <SkeletonList count={3} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">Stage Templates</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const stages = data?.items || [];
  const sortedStages = [...stages].sort((a, b) => a.order - b.order);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Stage Templates</h1>
          <p className="text-muted-foreground mt-1">
            Define the stages for your application process
          </p>
        </div>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          Create Stage
        </Button>
      </div>

      {sortedStages.length === 0 ? (
        <EmptyState
          icon={<ListOrdered className="h-12 w-12" />}
          title="No stage templates yet"
          description="Create stage templates like 'Applied', 'Phone Screen', 'Interview', 'Offer' to track your application progress"
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              Create Stage
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
