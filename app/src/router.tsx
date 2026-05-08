import { createBrowserRouter } from "react-router-dom";
import Home from "./routes/Home.tsx";
import NotFound from "./routes/NotFound.tsx";
import Auth from "./routes/Auth.tsx";
import React, { Suspense } from "react";
import { useDeeptrain } from "@/conf/env.ts";
import Register from "@/routes/Register.tsx";
import Forgot from "@/routes/Forgot.tsx";
import { lazyFactor } from "@/utils/loader.tsx";
import Index from "@/routes/Index.tsx";
import {
  AdminRequired,
  AuthForbidden,
  AuthRequired,
} from "@/routes/RouteGuards.tsx";
import {
  AdminContentSkeleton,
  AdminShellSkeleton,
} from "@/components/admin/AdminSkeleton.tsx";

const modelRoute = lazyFactor(() => import("@/routes/Model.tsx"));
const personalizationRoute = lazyFactor(
  () => import("@/routes/Personalization.tsx"),
);
const walletRoute = lazyFactor(() => import("@/routes/Wallet.tsx"));
const accountRoute = lazyFactor(() => import("@/routes/Account.tsx"));

const generationRoute = lazyFactor(() => import("@/routes/Generation.tsx"));
const sharingRoute = lazyFactor(() => import("@/routes/Sharing.tsx"));
const articleRoute = lazyFactor(() => import("@/routes/Article.tsx"));

const adminPageRoute = lazyFactor(() => import("@/routes/Admin.tsx"));
const adminDashboardRoute = lazyFactor(
  () => import("@/routes/admin/DashBoard.tsx"),
);
const adminMarketRoute = lazyFactor(() => import("@/routes/admin/Market.tsx"));
const adminChannelRoute = lazyFactor(() => import("@/routes/admin/Channel.tsx"));
const adminSystemRoute = lazyFactor(() => import("@/routes/admin/System.tsx"));
const adminChargeRoute = lazyFactor(() => import("@/routes/admin/Charge.tsx"));
const adminUsersRoute = lazyFactor(() => import("@/routes/admin/Users.tsx"));
const adminBroadcastRoute = lazyFactor(
  () => import("@/routes/admin/Broadcast.tsx"),
);
const adminSubscriptionRoute = lazyFactor(
  () => import("@/routes/admin/Subscription.tsx"),
);
const adminAttachmentRoute = lazyFactor(
  () => import("@/routes/admin/Attachment.tsx"),
);
const adminLoggerRoute = lazyFactor(() => import("@/routes/admin/Logger.tsx"));
const adminRecordRoute = lazyFactor(() => import("@/routes/admin/Record.tsx"));
const adminPaymentRoute = lazyFactor(() => import("@/routes/admin/Payment.tsx"));
const adminWarmupRoute = lazyFactor(() => import("@/routes/admin/Warmup.tsx"));

const withSuspense = (
  component: React.ComponentType,
  fallback: React.ReactNode = null,
) =>
  React.createElement(
    Suspense,
    { fallback },
    React.createElement(component),
  );

const router = createBrowserRouter([
  {
    id: "index",
    path: "/",
    Component: Index,
    ErrorBoundary: NotFound,
    children: [
      {
        id: "not-found",
        path: "*",
        element: <NotFound />,
      },
      {
        id: "home",
        path: "",
        element: <Home />,
      },
      {
        id: "personalization",
        path: "personalization",
        element: withSuspense(personalizationRoute),
      },
      {
        id: "model",
        path: "model",
        element: withSuspense(modelRoute),
      },
      {
        id: "wallet",
        path: "wallet",
        element: withSuspense(walletRoute),
      },
      // {
      //   id: "log",
      //   path: "log",
      //   element: (
      //     <Suspense>
      //       <License />
      //     </Suspense>
      //   ),
      // },
      // {
      //   id: "preset",
      //   path: "preset",
      //   element: (
      //     <Suspense>
      //       <Preset />
      //     </Suspense>
      //   ),
      // },
      // {
      //   id: "key",
      //   path: "key",
      //   element: (
      //     <Suspense>
      //       <License />
      //     </Suspense>
      //   ),
      // },
      {
        id: "account",
        path: "account",
        element: withSuspense(accountRoute),
      },
      {
        id: "login",
        path: "/login",
        element: (
          <AuthForbidden>
            <Auth />
          </AuthForbidden>
        ),
        ErrorBoundary: NotFound,
      },
      {
        id: "admin",
        path: "/admin",
        element: (
          <AdminRequired>
            {withSuspense(adminPageRoute, <AdminShellSkeleton />)}
          </AdminRequired>
        ),
        children: [
          {
            id: "admin-dashboard",
            path: "",
            element: withSuspense(adminDashboardRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-users",
            path: "users",
            element: withSuspense(adminUsersRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-market",
            path: "market",
            element: withSuspense(adminMarketRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-channel",
            path: "channel",
            element: withSuspense(adminChannelRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-system",
            path: "system",
            element: withSuspense(adminSystemRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-attachment",
            path: "attachment",
            element: withSuspense(adminAttachmentRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-warm-up",
            path: "warmup",
            element: withSuspense(adminWarmupRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-charge",
            path: "charge",
            element: withSuspense(adminChargeRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-broadcast",
            path: "broadcast",
            element: withSuspense(adminBroadcastRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-subscription",
            path: "subscription",
            element: withSuspense(
              adminSubscriptionRoute,
              <AdminContentSkeleton />,
            ),
          },
          {
            id: "admin-record",
            path: "record",
            element: withSuspense(adminRecordRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-payment",
            path: "pay",
            element: withSuspense(adminPaymentRoute, <AdminContentSkeleton />),
          },
          {
            id: "admin-logger",
            path: "logger",
            element: withSuspense(adminLoggerRoute, <AdminContentSkeleton />),
          },
        ],
        ErrorBoundary: NotFound,
      },
      {
        id: "generation",
        path: "/generate",
        element: (
          <AuthRequired>
            {withSuspense(generationRoute)}
          </AuthRequired>
        ),
        ErrorBoundary: NotFound,
      },
      {
        id: "article",
        path: "/article",
        element: (
          <AuthRequired>
            {withSuspense(articleRoute)}
          </AuthRequired>
        ),
        ErrorBoundary: NotFound,
      },
      ...(useDeeptrain
        ? []
        : [
            {
              id: "register",
              path: "/register",
              element: (
                <AuthForbidden>
                  <Register />
                </AuthForbidden>
              ),
              ErrorBoundary: NotFound,
            },
            {
              id: "forgot",
              path: "/forgot",
              element: (
                <AuthForbidden>
                  <Forgot />
                </AuthForbidden>
              ),
              ErrorBoundary: NotFound,
            },
          ]),
    ],
  },
  {
    id: "share",
    path: "/share/:hash",
    element: withSuspense(sharingRoute),
    ErrorBoundary: NotFound,
  },
]);

export default router;
