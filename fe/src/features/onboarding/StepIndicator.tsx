import { cn } from '@/shared/lib/utils';

interface StepIndicatorProps {
  currentStep: number;
  totalSteps: number;
}

export function StepIndicator({ currentStep, totalSteps }: StepIndicatorProps) {
  return (
    <div className="flex items-center gap-1.5">
      {Array.from({ length: totalSteps }, (_, i) => (
        <div
          key={i}
          className={cn(
            'h-2 rounded-full transition-all duration-300',
            i === currentStep
              ? 'w-6 bg-primary'
              : 'w-2 bg-muted-foreground/30'
          )}
        />
      ))}
    </div>
  );
}
