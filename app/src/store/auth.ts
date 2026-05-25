import { createSlice } from "@reduxjs/toolkit";
import axios from "axios";
import { tokenField } from "@/conf/bootstrap.ts";
import { AppDispatch, RootState } from "./index.ts";
import { forgetMemory, setMemory } from "@/utils/memory.ts";
import { doState } from "@/api/auth.ts";

export const authSlice = createSlice({
  name: "auth",
  initialState: {
    token: "",
    init: false,
    authenticated: false,
    admin: false,
    username: "",
    tasks: [] as number[],
  },
  reducers: {
    setToken: (state, action) => {
      const token = (action.payload as string).trim();
      state.token = token;
      axios.defaults.headers.common["Authorization"] = token;
      if (token.length > 0) setMemory(tokenField, token);
      else forgetMemory(tokenField);
    },
    setAuthenticated: (state, action) => {
      state.authenticated = action.payload as boolean;
    },
    setUsername: (state, action) => {
      state.username = action.payload as string;
    },
    setInit: (state, action) => {
      state.init = action.payload as boolean;
    },
    setAdmin: (state, action) => {
      state.admin = action.payload as boolean;
    },
    updateData: (state, action) => {
      state.init = true;
      state.authenticated = action.payload.authenticated as boolean;
      state.username = action.payload.username as string;
      state.admin = action.payload.admin as boolean;
    },
    increaseTask: (state, action) => {
      state.tasks.push(action.payload as number);
    },
    decreaseTask: (state, action) => {
      state.tasks = state.tasks.filter((v) => v !== (action.payload as number));
    },
    clearTask: (state) => {
      state.tasks = [];
    },
    logout: (state) => {
      state.token = "";
      state.authenticated = false;
      state.username = "";
      state.admin = false;
      axios.defaults.headers.common["Authorization"] = "";
      forgetMemory(tokenField);
    },
  },
});

function clearInvalidToken(dispatch: AppDispatch) {
  dispatch(authSlice.actions.logout());
  dispatch(
    authSlice.actions.updateData({
      authenticated: false,
      username: "",
      admin: false,
    }),
  );
}

function keepTokenForRetry(dispatch: AppDispatch) {
  dispatch(
    authSlice.actions.updateData({
      authenticated: false,
      username: "",
      admin: false,
    }),
  );
}

export function validateToken(
  dispatch: AppDispatch,
  token: string,
  hook?: () => void,
) {
  token = token.trim();

  if (token.length === 0) {
    clearInvalidToken(dispatch);
    return;
  }

  dispatch(setToken(token));
  doState()
    .then((data) => {
      if (!data.status) {
        clearInvalidToken(dispatch);
        return;
      }

      dispatch(
        updateData({
          authenticated: true,
          username: data.user,
          admin: data.admin,
        }),
      );

      hook && hook();
    })
    .catch((err) => {
      console.debug(err);
      keepTokenForRetry(dispatch);
    });
}

export const selectAuthenticated = (state: RootState) =>
  state.auth.authenticated;
export const selectUsername = (state: RootState) => state.auth.username;
export const selectInit = (state: RootState) => state.auth.init;
export const selectAdmin = (state: RootState) => state.auth.admin;
export const selectTasks = (state: RootState) => state.auth.tasks;
export const selectTasksLength = (state: RootState) => state.auth.tasks.length;
export const selectIsTasking = (state: RootState) =>
  state.auth.tasks.length > 0;

export const {
  setToken,
  setAuthenticated,
  setUsername,
  logout,
  setInit,
  setAdmin,
  updateData,
  increaseTask,
  decreaseTask,
  clearTask,
} = authSlice.actions;
export default authSlice.reducer;
