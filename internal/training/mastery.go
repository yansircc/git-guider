package training

// Mastery algorithm: 2 consecutive passes to master a topic.
// Any failure resets consecutive_passes to 0.
// mastered=1 persists once earned.

const MasteryThreshold = 2
