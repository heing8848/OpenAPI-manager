import { Container, Box } from '@mui/material';
import React from 'react';
import { useSelector } from 'react-redux';

const Footer = () => {
  const siteInfo = useSelector((state) => state.siteInfo);

  return (
    <Container
      sx={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '64px',
      }}
    >
      <Box sx={{ textAlign: 'center' }}>
        {siteInfo.footer_html ? (
          <div
            className="custom-footer"
            dangerouslySetInnerHTML={{ __html: siteInfo.footer_html }}
          ></div>
        ) : (
          <>Alfred API项目参考GitHub开源项目One API (One API ) 由 Alfred改写功能.</>
        )}
      </Box>
    </Container>
  );
};

export default Footer;
