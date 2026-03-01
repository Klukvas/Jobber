import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { companiesService } from "@/services/companiesService";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { SkeletonList } from "@/shared/ui/Skeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ErrorState } from "@/shared/ui/ErrorState";
import {
  Plus,
  Building2,
  MapPin,
  MoreVertical,
  Edit,
  Trash2,
  Briefcase,
  Clock,
  ArrowUp,
  ArrowDown,
  Heart,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { CreateCompanyModal } from "@/features/companies/modals/CreateCompanyModal";
import { DeleteCompanyDialog } from "@/features/companies/modals/DeleteCompanyDialog";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import type { CompanyDTO } from "@/shared/types/api";

type SortBy = "name" | "last_activity" | "applications_count";
type SortDir = "asc" | "desc";

export default function Companies() {
  const { t } = useTranslation();
  usePageMeta({ titleKey: "companies.title", noindex: true });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingCompany, setEditingCompany] = useState<CompanyDTO | null>(null);
  const [deletingCompany, setDeletingCompany] = useState<CompanyDTO | null>(
    null,
  );
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [sortBy, setSortBy] = useState<SortBy>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  // Close context menu when clicking outside
  useEffect(() => {
    if (!openMenuId) return;
    const handleClickOutside = () => setOpenMenuId(null);
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [openMenuId]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["companies", sortBy, sortDir],
    queryFn: () =>
      companiesService.list({
        limit: 100,
        offset: 0,
        sort_by: sortBy,
        sort_dir: sortDir,
      }),
    staleTime: 30_000,
  });

  const toggleFavoriteMutation = useMutation({
    mutationFn: companiesService.toggleFavorite,
    onMutate: async (companyId) => {
      const queryKey = ["companies", sortBy, sortDir];
      await queryClient.cancelQueries({ queryKey: ["companies"] });
      const previous = queryClient.getQueryData(queryKey);
      queryClient.setQueryData(queryKey, (old: typeof data) => {
        if (!old) return old;
        return {
          ...old,
          items: old.items.map((c: CompanyDTO) =>
            c.id === companyId ? { ...c, is_favorite: !c.is_favorite } : c,
          ),
        };
      });
      return { previous, queryKey };
    },
    onError: (_err, _companyId, context) => {
      if (context?.previous && context?.queryKey) {
        queryClient.setQueryData(context.queryKey, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["companies"] });
    },
  });

  const toggleSort = (field: SortBy) => {
    if (sortBy === field) {
      setSortDir(sortDir === "desc" ? "asc" : "desc");
    } else {
      setSortBy(field);
      setSortDir(field === "name" ? "asc" : "desc");
    }
  };

  const handleEdit = (company: CompanyDTO) => {
    setEditingCompany(company);
    setOpenMenuId(null);
  };

  const handleDelete = (company: CompanyDTO) => {
    setDeletingCompany(company);
    setOpenMenuId(null);
  };

  const handleViewApplications = (companyId: string) => {
    navigate(`/app/applications?company_id=${companyId}`);
  };

  const getCompanyStatusDisplay = (status: string) => {
    switch (status) {
      case "idle":
        return {
          label: t("companies.statusIdle"),
          className:
            "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400",
        };
      case "active":
        return {
          label: t("companies.statusActive"),
          className:
            "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
        };
      case "interviewing":
        return {
          label: t("companies.statusInterviewing"),
          className:
            "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
        };
      default:
        return {
          label: status,
          className:
            "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400",
        };
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">{t("companies.title")}</h1>
        </div>
        <SkeletonList count={3} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t("companies.title")}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const companies = data?.items || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t("companies.title")}</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("companies.create")}
        </Button>
      </div>

      {companies.length === 0 ? (
        <EmptyState
          icon={<Building2 className="h-12 w-12" />}
          title={t("companies.noCompanies")}
          description={t("companies.createFirst")}
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              {t("companies.create")}
            </Button>
          }
        />
      ) : (
        <>
          {/* Sorting Controls */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-muted-foreground">
              {t("common.sortBy")}
            </span>
            <Button
              variant={sortBy === "name" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("name")}
            >
              <Building2 className="h-3 w-3 mr-1" />
              {t("companies.sortName")}
              {sortBy === "name" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
            <Button
              variant={sortBy === "last_activity" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("last_activity")}
            >
              <Clock className="h-3 w-3 mr-1" />
              {t("companies.sortLastActivity")}
              {sortBy === "last_activity" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
            <Button
              variant={sortBy === "applications_count" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("applications_count")}
            >
              <Briefcase className="h-3 w-3 mr-1" />
              {t("companies.sortApplications")}
              {sortBy === "applications_count" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
          </div>

          {/* Company Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {companies.map((company) => {
              const statusDisplay = getCompanyStatusDisplay(
                company.derived_status,
              );
              return (
                <div key={company.id} className="relative">
                  <Card className="transition-all hover:shadow-md h-full group">
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between gap-2">
                        <CardTitle className="text-xl font-bold leading-tight flex-1">
                          {company.name}
                        </CardTitle>
                        <div className="flex items-center gap-1">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              e.preventDefault();
                              toggleFavoriteMutation.mutate(company.id);
                            }}
                            disabled={toggleFavoriteMutation.isPending}
                            className="p-1 rounded-md hover:bg-accent transition-colors disabled:opacity-50"
                            aria-label={
                              company.is_favorite
                                ? t("common.removeFromFavorites")
                                : t("common.addToFavorites")
                            }
                          >
                            <Heart
                              className={`h-4 w-4 ${company.is_favorite ? "fill-red-500 text-red-500" : "text-muted-foreground"}`}
                            />
                          </button>
                          <div
                            className="relative"
                            onClick={(e) => e.preventDefault()}
                          >
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                e.preventDefault();
                                setOpenMenuId(
                                  openMenuId === company.id ? null : company.id,
                                );
                              }}
                              className="p-1 rounded-md hover:bg-accent transition-colors text-muted-foreground"
                              aria-label="Company actions"
                            >
                              <MoreVertical className="h-4 w-4" />
                            </button>
                            {openMenuId === company.id && (
                              <div className="absolute right-0 mt-1 w-40 bg-popover border rounded-md shadow-lg z-10">
                                <button
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    handleEdit(company);
                                  }}
                                  className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                                >
                                  <Edit className="h-4 w-4" />
                                  {t("common.edit")}
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    handleDelete(company);
                                  }}
                                  className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left text-destructive"
                                >
                                  <Trash2 className="h-4 w-4" />
                                  {t("common.delete")}
                                </button>
                              </div>
                            )}
                          </div>
                        </div>
                      </div>

                      {company.location && (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                          <MapPin className="h-4 w-4" />
                          <span>{company.location}</span>
                        </div>
                      )}
                    </CardHeader>

                    <CardContent className="space-y-3 pt-0">
                      {/* Status Badge */}
                      <div className="flex items-center justify-between">
                        <span
                          className={`inline-flex items-center rounded-full font-medium text-sm px-2.5 py-1 ${statusDisplay.className}`}
                        >
                          {statusDisplay.label}
                        </span>
                      </div>

                      {/* Statistics */}
                      {company.applications_count > 0 ? (
                        <div className="space-y-2 text-sm border-t pt-3">
                          <div className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {t("companies.totalApplications")}
                            </span>
                            <span className="font-medium">
                              {company.applications_count}
                            </span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-muted-foreground">
                              {t("companies.activeApplications")}
                            </span>
                            <span className="font-medium">
                              {company.active_applications_count}
                            </span>
                          </div>
                          {company.last_activity_at && (
                            <div className="flex items-center gap-2 text-muted-foreground">
                              <Clock className="h-3.5 w-3.5" />
                              <span>
                                {t("companies.lastActivity")}{" "}
                                {formatDistanceToNow(
                                  new Date(company.last_activity_at),
                                  {
                                    addSuffix: true,
                                  },
                                )}
                              </span>
                            </div>
                          )}
                        </div>
                      ) : (
                        <div className="text-sm text-muted-foreground border-t pt-3 text-center py-2">
                          {t("companies.noApplicationsYet")}
                        </div>
                      )}

                      {/* Notes */}
                      {company.notes && (
                        <p className="text-sm text-muted-foreground line-clamp-2">
                          {company.notes}
                        </p>
                      )}

                      {/* Quick Actions */}
                      {company.applications_count > 0 && (
                        <Button
                          variant="outline"
                          size="sm"
                          className="w-full"
                          onClick={() => handleViewApplications(company.id)}
                        >
                          <Briefcase className="h-3.5 w-3.5 mr-1.5" />
                          {t("companies.viewApplications", {
                            count: company.applications_count,
                          })}
                        </Button>
                      )}
                    </CardContent>
                  </Card>
                </div>
              );
            })}
          </div>
        </>
      )}

      {/* Modals */}
      <CreateCompanyModal
        open={isCreateModalOpen || !!editingCompany}
        onOpenChange={(open) => {
          setIsCreateModalOpen(open);
          if (!open) setEditingCompany(null);
        }}
        company={editingCompany || undefined}
      />

      {deletingCompany && (
        <DeleteCompanyDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setDeletingCompany(null);
          }}
          company={deletingCompany}
        />
      )}
    </div>
  );
}
