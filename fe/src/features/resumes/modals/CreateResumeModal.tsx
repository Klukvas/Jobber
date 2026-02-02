import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { resumesService } from '@/services/resumesService';
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
import { showErrorNotification, showSuccessNotification } from '@/shared/lib/notifications';

interface CreateResumeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type UploadMode = 'url' | 'file';

export function CreateResumeModal({ open, onOpenChange }: CreateResumeModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  
  const [mode, setMode] = useState<UploadMode>('url');
  const [title, setTitle] = useState('');
  const [fileUrl, setFileUrl] = useState('');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);

  // Traditional URL-based resume creation
  const createMutation = useMutation({
    mutationFn: resumesService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resumes'] });
      showSuccessNotification('Resume created successfully');
      resetAndClose();
    },
    onError: (error: any) => {
      showErrorNotification(error?.message || 'Failed to create resume');
    },
  });

  // File upload mutation
  const uploadMutation = useMutation({
    mutationFn: async (file: File) => {
      const resume = await resumesService.uploadResume(file, setUploadProgress);
      // Update title if provided
      if (title && title !== 'Untitled Resume') {
        return resumesService.update(resume.id, { title, is_active: true });
      }
      return resumesService.update(resume.id, { is_active: true });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resumes'] });
      showSuccessNotification('Resume uploaded successfully');
      resetAndClose();
    },
    onError: (error: any) => {
      showErrorNotification(error?.message || 'Failed to upload resume');
      setUploadProgress(0);
    },
  });

  const resetAndClose = () => {
    onOpenChange(false);
    setTimeout(() => {
      setTitle('');
      setFileUrl('');
      setSelectedFile(null);
      setUploadProgress(0);
      setMode('url');
    }, 300);
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      // Validate file type
      if (file.type !== 'application/pdf') {
        showErrorNotification('Only PDF files are allowed');
        e.target.value = '';
        return;
      }
      // Validate file size (max 10MB)
      if (file.size > 10 * 1024 * 1024) {
        showErrorNotification('File size must be less than 10MB');
        e.target.value = '';
        return;
      }
      setSelectedFile(file);
      // Auto-fill title from filename if empty
      if (!title) {
        const fileName = file.name.replace(/\.[^/.]+$/, ''); // Remove extension
        setTitle(fileName);
      }
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (mode === 'url') {
      if (title && fileUrl) {
        createMutation.mutate({ title, file_url: fileUrl, is_active: true });
      }
    } else {
      if (selectedFile) {
        uploadMutation.mutate(selectedFile);
      }
    }
  };

  const isLoading = createMutation.isPending || uploadMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={resetAndClose}>
        <DialogHeader>
          <DialogTitle>{t('resumes.create')}</DialogTitle>
          <DialogDescription>Add a new resume to your collection</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            {/* Mode Selection - Switch Toggle */}
            <div className="space-y-2">
              <Label>Upload Method</Label>
              <div className="flex items-center gap-3 p-1 bg-muted rounded-lg w-fit">
                <button
                  type="button"
                  onClick={() => setMode('url')}
                  disabled={isLoading}
                  className={`px-4 py-2 text-sm font-medium rounded-md transition-all ${
                    mode === 'url'
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  External URL
                </button>
                <button
                  type="button"
                  onClick={() => setMode('file')}
                  disabled={isLoading}
                  className={`px-4 py-2 text-sm font-medium rounded-md transition-all ${
                    mode === 'file'
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  Upload PDF File
                </button>
              </div>
            </div>

            {/* Title Field */}
            <div className="space-y-2">
              <Label htmlFor="title">
                Title {mode === 'url' ? '*' : ''}
              </Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g., Software Engineer Resume - 2024"
                required={mode === 'url'}
                disabled={isLoading}
              />
              {mode === 'file' && (
                <p className="text-xs text-gray-500">
                  Optional: Auto-filled from filename, you can edit after upload
                </p>
              )}
            </div>

            {/* URL Mode */}
            {mode === 'url' && (
              <div className="space-y-2">
                <Label htmlFor="fileUrl">File URL *</Label>
                <Input
                  id="fileUrl"
                  type="url"
                  value={fileUrl}
                  onChange={(e) => setFileUrl(e.target.value)}
                  placeholder="https://example.com/my-resume.pdf"
                  required
                  disabled={isLoading}
                />
                <p className="text-xs text-gray-500">
                  Link to your resume on Google Drive, Dropbox, etc.
                </p>
              </div>
            )}

            {/* File Upload Mode */}
            {mode === 'file' && (
              <div className="space-y-2">
                <Label htmlFor="file">PDF File *</Label>
                <Input
                  id="file"
                  type="file"
                  accept="application/pdf,.pdf"
                  onChange={handleFileChange}
                  required
                  disabled={isLoading}
                  className="cursor-pointer"
                />
                {selectedFile && (
                  <div className="text-sm text-gray-600">
                    <p>Selected: {selectedFile.name}</p>
                    <p className="text-xs">Size: {(selectedFile.size / 1024).toFixed(2)} KB</p>
                  </div>
                )}
                <p className="text-xs text-gray-500">
                  Only PDF files, max 10MB
                </p>
              </div>
            )}

            {/* Upload Progress */}
            {uploadMutation.isPending && uploadProgress > 0 && (
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>Uploading...</span>
                  <span>{uploadProgress}%</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                    style={{ width: `${uploadProgress}%` }}
                  />
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={resetAndClose}
              disabled={isLoading}
            >
              {t('common.cancel')}
            </Button>
            <Button 
              type="submit" 
              disabled={
                isLoading || 
                (mode === 'url' && (!title || !fileUrl)) ||
                (mode === 'file' && !selectedFile)
              }
            >
              {isLoading 
                ? (uploadMutation.isPending ? 'Uploading...' : t('common.loading'))
                : mode === 'file' 
                  ? 'Upload' 
                  : t('common.create')
              }
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
