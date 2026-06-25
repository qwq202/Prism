import { createSlice } from "@reduxjs/toolkit";
import { phone } from "@/utils/device.ts";
import { RootState } from "@/store/index.ts";

export const menuSlice = createSlice({
  name: "menu",
  initialState: {
    open: !phone, // phone: false, tablet/desktop: true
  },
  reducers: {
    toggleMenu: (state) => {
      state.open = !state.open;
    },
    closeMenu: (state) => {
      state.open = false;
    },
    openMenu: (state) => {
      state.open = true;
    },
    setMenu: (state, action) => {
      state.open = action.payload as boolean;
    },
  },
});

export const { toggleMenu, closeMenu, openMenu, setMenu } = menuSlice.actions;
export default menuSlice.reducer;

export const selectMenu = (state: RootState): boolean => state.menu.open;
