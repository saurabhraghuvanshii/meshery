import React, { useEffect, useState } from "react";
import {
  Box,
  styled,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  SvgIcon,
} from "@sistent/sistent";

const CARD_WIDTH_PX = 445;
const HEADER_HEIGHT_PX = 64;
const CONTENT_HEIGHT_PX = 149;
const CONTENT_TOP_PX = 42;
const CONTENT_BOTTOM_PX = 30;
const CONTENT_SIDE_PX = 24;
const STRIP_HEIGHT_PX = 32;
const SAFFRON = "#E8C11A";
const CONTENT_TEXT_COLOR = "#3F444A";

const SessionExpiredContent = styled(Box)(() => ({
  width: "100%",
  minHeight: `${CONTENT_HEIGHT_PX}px`,
  overflowWrap: "break-word",
  textAlign: "center",
  padding: `${CONTENT_TOP_PX}px ${CONTENT_SIDE_PX}px ${CONTENT_BOTTOM_PX}px ${CONTENT_SIDE_PX}px`,
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  backgroundColor: "#F5F5F5",
}));

const IconContainer = styled(Box)(() => ({
  width: "28px",
  height: "28px",
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  position: "absolute",
  left: "16px",
  top: "50%",
  transform: "translateY(-50%)",
  borderRadius: "16px",
  color: "#ffffff",
}));

const RoundedWarningIcon = () => (
  <SvgIcon viewBox="0 0 24 24" sx={{ width: 26, height: 26 }}>
    <path
      fill="#ffffff"
      d="M11.02 3.95c.44-.76 1.52-.76 1.96 0l8.62 14.94c.44.76-.11 1.71-.98 1.71H3.38c-.87 0-1.42-.95-.98-1.71L11.02 3.95Z"
      style={{ strokeLinejoin: "round" }}
    />
    <path
      fill={SAFFRON}
      d="M11 8.25c0-.55.45-1 1-1s1 .45 1 1v5.1c0 .55-.45 1-1 1s-1-.45-1-1v-5.1Z"
    />
    <path
      fill={SAFFRON}
      d="M12 17.55a1.15 1.15 0 1 0 0-2.3 1.15 1.15 0 0 0 0 2.3Z"
    />
  </SvgIcon>
);

function AlertUnauthenticatedSession() {
  const [countDown, setCountDown] = useState(3);

  useEffect(() => {
    const timer = setTimeout(() => {
      if (countDown === 1) {
        // Propagate existing request parameters, if present.
        const existingQueryString = window.location.search;
        window.location = `/user/login${existingQueryString}`;
        return;
      }
      setCountDown((countDown) => countDown - 1);
    }, 1000);
    return () => clearTimeout(timer);
  }, [countDown]);

  return (
    <Dialog
      open
      disableEscapeKeyDown
      aria-labelledby="alert-dialog-title"
      aria-describedby="alert-dialog-description"
      BackdropProps={{
        style: {
          backgroundColor: "rgba(0, 0, 0, 0.58)",
        },
      }}
      PaperProps={{
        style: {
          width: "calc(100% - 32px)",
          maxWidth: `${CARD_WIDTH_PX}px`,
          borderBottom: `${STRIP_HEIGHT_PX}px solid ${SAFFRON}`,
          borderRadius: "6px",
          overflow: "hidden",
          boxShadow: "0 18px 42px rgba(0, 0, 0, 0.35)",
        },
      }}
    >
      <DialogTitle
        id="alert-dialog-title"
        sx={{
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          position: "relative",
          textAlign: "center",
          padding: "0 16px",
          color: "#ffffff",
          backgroundColor: SAFFRON,
          fontSize: "18px",
          lineHeight: "26px",
          fontWeight: 500,
          minHeight: `${HEADER_HEIGHT_PX}px`,
        }}
      >
        <IconContainer>
          <RoundedWarningIcon />
        </IconContainer>
        Session Expired
      </DialogTitle>
      <DialogContent sx={{ padding: 0 }}>
        <SessionExpiredContent id="alert-dialog-description">
          <Typography sx={{ color: CONTENT_TEXT_COLOR, fontSize: "15px", marginBottom: "16px" }}>
            User not authenticated
          </Typography>
          <Typography sx={{ color: CONTENT_TEXT_COLOR, fontSize: "15px" }}>
            You will be redirected to Login page in {countDown}
          </Typography>
        </SessionExpiredContent>
      </DialogContent>
    </Dialog>
  );
}

export default AlertUnauthenticatedSession;
