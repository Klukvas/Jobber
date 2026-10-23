import { useTranslation } from "react-i18next";
import { ProfessionalTemplate } from "./ProfessionalTemplate";
import { MobileTextEditingContext } from "./mobileTextEditing";
import "./mobileTextEditor.css";

/** The document's existing inline fields in a readable, fluid presentation. */
export function MobileTextEditor() {
  const { t } = useTranslation();
  return (
    <MobileTextEditingContext.Provider value={true}>
      <div className="mobile-resume-text m-3 min-w-0 rounded-lg border bg-white p-4 text-black">
        <p className="mb-5 text-sm text-gray-600">{t("resumeBuilder.mobile.hint")}</p>
        <ProfessionalTemplate editable />
      </div>
    </MobileTextEditingContext.Provider>
  );
}
