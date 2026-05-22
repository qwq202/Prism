import "@/assets/admin/all.less";
import MenuBar from "@/components/admin/MenuBar.tsx";
import { useLocation, useOutlet } from "react-router-dom";
import { useSelector } from "react-redux";
import { selectAdmin, selectInit } from "@/store/auth.ts";
import { useEffect } from "react";
import router from "@/router.tsx";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { AnimatePresence, motion } from "framer-motion";

const adminMenuVariants = {
  hidden: { opacity: 0, x: -16 },
  visible: {
    opacity: 1,
    x: 0,
    transition: { duration: 0.35, ease: "easeOut" },
  },
};

const adminContentVariants = {
  hidden: { opacity: 0, y: 14 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.38, ease: "easeOut" },
  },
  exit: {
    opacity: 0,
    y: -8,
    transition: { duration: 0.18, ease: "easeIn" },
  },
};

function Admin() {
  const init = useSelector(selectInit);
  const admin = useSelector(selectAdmin);
  const location = useLocation();
  const outlet = useOutlet();

  useEffect(() => {
    if (init && !admin) router.navigate("/");
  }, [init, admin]);

  return (
    <div className={`home-page flex flex-row flex-1`}>
      <div className={`admin-page`}>
        <motion.div
          className="admin-menu-motion"
          variants={adminMenuVariants}
          initial="hidden"
          animate="visible"
        >
          <MenuBar />
        </motion.div>
        <ScrollArea className={`admin-content`}>
          <AnimatePresence mode="wait">
            <motion.div
              className="admin-route-motion"
              key={location.pathname}
              variants={adminContentVariants}
              initial="hidden"
              animate="visible"
              exit="exit"
            >
              {outlet}
            </motion.div>
          </AnimatePresence>
        </ScrollArea>
      </div>
    </div>
  );
}

export default Admin;
