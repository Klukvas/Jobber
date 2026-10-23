import { createContext, useContext } from "react";
export const MobileTextEditingContext = createContext(false);
export const useMobileTextEditing = () => useContext(MobileTextEditingContext);
