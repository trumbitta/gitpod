/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

/**
 * `IdeServer` provides IDE related info like IDE preferences for the dashboard.
 */
export interface IdeServer {
    /**
     * Returns the IDE preferences.
     */
    getIdePreferences(): Promise<IdePreferences>;
}

/**
 * IDE preferences (texts for the dashboard as well as available IDE options).
 */
export interface IdePreferences {
    /**
     * Header text of the IDE preferences section (plain text only).
     */
    sectionTitle: string;
    /**
     * Subheader text of the IDE preferences section (plain text only).
     */
    sectionSubtitle: string;

    /**
     * Header text of the desktop IDE preferences subsection (plain text only).
     */
    desktopIdeTitle: string;
    /**
     * Subheader text of the desktop IDE preferences subsection (plain text
     * only).
     */
    desktopIdeSubtitle: string;
    /**
     * Text of an optional label next to the desktop IDE preferences header like
     * “Beta” (plain text only).
     */
    desktopIdeLabel?: string;
    /**
     * Text of an optional info box next to the desktop IDE preferences (HTML).
     */
    desktopIdeInfobox?: string;
    /**
     * Text of an optional footnote next to the desktop IDE preferences (HTML).
     */
    desktopIdeFootnote?: string;

    /**
     * The default IDE when the user has not specified one.
     */
    defaultIde: string;
    /**
     * The default desktop IDE when the user has not specified one.
     */
    defaultDesktopIde: string;

    /**
     * A list of available IDEs.
     */
    options: { [key: string]: IdeOption };
}

export interface IdeOption {
    /**
     * Human readable title text of the IDE (plain text only).
     */
    title: string;
    /**
     * The type of the IDE, currently 'browser' or 'desktop'.
     */
    type: 'browser' | 'desktop';
    /**
     * The logo for the IDE. That could be a key in (see
     * components/dashboard/src/images/ideLogos.ts) or a URL.
     */
    logo: string;
    /**
     * Text of an optional tooltip (plain text only).
     */
    tooltip?: string;
    /**
     * Text of an optional label next to the IDE option like “Insiders” (plain
     * text only).
     */
    label?: string;
    /**
     * If `true` this IDE option is not visible in the IDE preferences.
     */
    hidden?: boolean;
    /**
     * The image ref to the IDE image.
     */
    image: string;
    /**
     * When this is `true`, the tag of this image is resolved to the latest
     * image digest regularly.
     *
     * This is useful if this image points to a tag like `nightly` that will be
     * updated regularly. When `resolveImageDigest` is `true`, we make sure that
     * we resolve the tag regularly to the most recent image version.
     */
    resolveImageDigest?: boolean;
    /**
     * Specify the order by setting an order key.
     */
    orderKey?: string;
}
