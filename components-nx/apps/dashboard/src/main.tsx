/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

import { StrictMode } from 'react';
import * as ReactDOM from 'react-dom';

import { BrowserRouter } from 'react-router-dom';

import { TeamsContextProvider } from '@components-nx/dashboard/features/teams';

// Components
import { App } from './app/app.component';

// Context
import { UserContextProvider } from './app/user-context';
import { ThemeContextProvider } from './app/theme-context';

// import './index.css';

ReactDOM.render(
  <StrictMode>
    <UserContextProvider>
      <TeamsContextProvider>
        <ThemeContextProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </ThemeContextProvider>
      </TeamsContextProvider>
    </UserContextProvider>
  </StrictMode>,
  document.getElementById('root')
);
